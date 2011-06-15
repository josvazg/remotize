// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't have to rewrite an interface by
// hand in order to use it remotely or out-of-process. 
package remotize

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"reflect"
	"rpc"
	"strconv"
	"strings"
	"sync"
	"time"
)

// UNSET_TIMEOUT
const NoTimeout = 0

// Error handler interface
type ErrorHandling func(string, os.Error)

// Remotized client type
type ClientBase struct {
	client  *rpc.Client   // rpc transport
	remote  interface{}   // local reference to remote interface
	Handler ErrorHandling // default error handler
	Timeout int64         // default rpc max timeout
}

// Remotized Client using the rpc package as transport
type Client interface {
	Bind(*rpc.Client)    // Binds this client to a rpc.Client
	Base() *ClientBase   // Gets a reference to the base ClientBase
	Remote() interface{} //  Obtain reference to the Remote Interface
}

// Remotized server type
type ServerBase struct {
	server *rpc.Server // rpc server
	impl   interface{} // iface implementation to be invoked
}

// Remotized Server using the rpc package as transport
type Server interface {
	Bind(*rpc.Server, interface{}) // Binds this server to a rpc.Server
	Base() *ServerBase             // Gets a reference to the base Srv
}

// Args
type Args struct {
	A []interface{}
}

// Results
type Results struct {
	R []interface{}
}

// Remotize Timeout
// Includes the timedout call in case the user still need to get it
type Timedout struct {
	os.Error
	Call *rpc.Call
}

// Pipe for local invocations, parent/child comms
type Pipe struct {
	in  io.ReadCloser
	out io.WriteCloser
}

// Remotized Registry
var reg = make(map[string]reflect.Type)

// Registry's lock
var lock sync.RWMutex

// Bind associates a ClientBase Client against a rpc.Client
func (c *ClientBase) Bind(client *rpc.Client) {
	c.client = client
}

// Gets a reference to the ClientBase configurable fields on ClientBase itself
func (c *ClientBase) Base() *ClientBase {
	return c
}

// Obtain the local reference to the Remote Interface
func (c *ClientBase) Remote() interface{} {
	return c.remote
}

// Bind associates a ServerBase Server against a rpc.Server and some implementation
func (s *ServerBase) Bind(server *rpc.Server, impl interface{}) {
	s.server = server
	s.impl = impl
}

// Gets a reference to the Base ServerBase 
func (s *ServerBase) Base() *ServerBase {
	return s
}

// Add a remotized type to the registry, so the interface is 'exported'.
func Register(c, s interface{}) {
	ct := reflect.TypeOf(c)
	st := reflect.TypeOf(s)
	cname := fmt.Sprintf("%v", ct)
	sname := fmt.Sprintf("%v", st)
	lock.Lock()
	reg[cname] = ct
	reg[sname] = st
	//fmt.Println("Registry is now", reg)
	lock.Unlock()
}

// Remove a type from registry, so the interface is 'unexported'.
func Unregister(name string) {
	lock.Lock()
	reg[name+"Client"] = nil, false
	reg[name+"Server"] = nil, false
	lock.Unlock()
}

// find from registry
func find(name string) reflect.Type {
	lock.RLock()
	defer lock.RLock()
	return reg[name]
}

// instatiante returns a Ptr instance of the given type, 
// if found in the registry, or nil otherwise
func instantiate(name string) interface{} {
	t := find(name)
	if t == nil {
		return nil
	}
	return reflect.New(t).Interface()
}

// nameFor returns the name of the given underliying type. Pointers are followed
// up to the final referenced type
func nameFor(i interface{}) string {
	t := reflect.TypeOf(i)
	for t.Kind() == reflect.Ptr {
		t = (t).Elem()
	}
	return fmt.Sprintf("%v", t)
}

// NewClient instantiates a client for the given interface, 
// if found on the registry, otherwise nil is returned.
// If pack is NOT empty that package name is used to locate the remotization,
// use it when the remotized code is on a different package from the 
// remotized interface
func NewClient(client *rpc.Client, i interface{}, pack string) Client {
	ifacename := nameFor(i)
	if pack != "" {
		dotpos := strings.LastIndex(ifacename, ".")
		ifacename = pack + "." + ifacename[dotpos:]
	}
	clt := instantiate(ifacename + "Client")
	if clt != nil {
		c := clt.(Client)
		c.Bind(client)
		return c
	}
	return nil
}

// NewServer instantiates a server for the given interface,
// if found on the registry, otherwise nil is returned.
// It also initiates it with the implementation of that interface
func NewServer(server *rpc.Server, i, impl interface{}, pack string) Server {
	if impl == nil {
		impl = i
	}
	ifacename := nameFor(i)
	if pack != "" {
		dotpos := strings.LastIndex(ifacename, ".")
		ifacename = pack + "." + ifacename[dotpos:]
	}
	srv := instantiate(ifacename + "Server")
	if srv != nil {
		s := srv.(Server)
		s.Bind(server, impl)
		return s
	}
	return nil
}

// Call to a remotized method
func Call(c *ClientBase, method string, args ...interface{}) (*Results, os.Error) {
	var r Results
	a := &Args{args}
	var e os.Error
	if c.Timeout == NoTimeout {
		e = c.client.Call(method, a, &r)
	} else {
		e = callTimeout(c, method, a, &r, c.Timeout)
	}
	return &r, e
}

// calltimeout calls with a timeout
func callTimeout(c *ClientBase, method string, args interface{},
reply interface{}, timeout int64) os.Error {
	call := c.client.Go(method, args, reply, nil)
	select {
	case <-call.Done:
		// Call returned
	case <-time.After(timeout):
		msg := fmt.Sprintf("Call timed out %vms at %v()!", timeout, method)
		return &Timedout{os.NewError(msg), call}
	}
	return call.Error
}

// HandleError handles a remote error. It will either call the preconfigured 
// client handler or just panic with the remote error.
func HandleError(c *ClientBase, funcname string, e os.Error) {
	if c.Handler != nil {
		c.Handler(funcname, e)
	} else {
		errmsg := fmt.Sprintf("Error at %v(): %v", funcname, e)
		panic(errmsg)
	}
}

// Read from the pipe
func (p *Pipe) Read(b []byte) (n int, err os.Error) {
	return p.in.Read(b)
}

// Write to the pipe
func (p *Pipe) Write(b []byte) (n int, err os.Error) {
	return p.out.Write(b)
}

// Close pipe io
func (p *Pipe) Close() os.Error {
	err := p.in.Close()
	if err != nil {
		return err
	}
	return p.out.Close()
}

// Prepare a ReadWriteCloser pipe from a reader and a writer
// This can be passed to NewClient to use RPCs over local pipe streams
func IO(in io.ReadCloser, out io.WriteCloser) *Pipe {
	return &Pipe{in, out}
}

// Old code
//------------------------------------------
// New code

func Remotize0(i interface{}) os.Error {
	/*if src, ok := i.([]ast.Decl); ok {
		return remotize0(src)
	}*/
	var t reflect.Type
	if _, ok := i.(reflect.Type); ok {
		t = i.(reflect.Type)
	} else {
		t = reflect.TypeOf(i)
	}
	if t.Kind() == reflect.Interface || t.NumMethod() > 0 {
		remotize0(declare(t))
		return nil
	}
	if t.Kind() == reflect.Ptr {
		return Remotize0(t.Elem())
	}
	// TODO error
	return nil
}

func declare(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Interface:
		return "type " + t.Name() + " interface" + methods(t)
	}
	st := t
	for ; st.Kind() == reflect.Ptr; st = t.Elem() {
	}
	return "type " + st.Name() + "er interface" + methods(t)
}

func source(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Array:
		return "[" + strconv.Itoa(t.Len()) + "]" + t.Name() + methods(t)
	case reflect.Chan:
		return "chan " + source(t.Elem()) + methods(t)
	case reflect.Func:
		return funcsource(t, nil)
	case reflect.Map:
		return "map[" + source(t.Key()) + "]" + source(t.Elem()) + methods(t)
	case reflect.Ptr:
		return "*" + source(t.Elem()) + methods(t)
	case reflect.Slice:
		return "[]" + source(t.Elem()) + methods(t)
	case reflect.String:
		return "string" + methods(t)
	}
	return t.String()
}

func methods(t reflect.Type) string {
	if t.NumMethod() > 0 {
		methods := " {"
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			methods += "\n" + funcsource(t, &m)
		}
		methods += "\n}"
		return methods
	}
	return ""
}

func funcsource(t reflect.Type, m *reflect.Method) string {
	fn := "func ("
	start := 0
	if t.Kind() == reflect.Interface {
		fn = m.Name + "("
	} else if m != nil && m.Name != "" {
		start++
		fn = m.Name + "("
	}
	if m != nil {
		t = m.Type
	}
	for i := start; i < t.NumIn(); i++ {
		fn += source(t.In(i))
		if (i + 1) != t.NumIn() {
			fn += ", "
		}
	}
	fn += ") "
	if t.NumOut() > 1 {
		fn += "("
	}
	for i := 0; i < t.NumOut(); i++ {
		fn += source(t.Out(i))
		if (i + 1) != t.NumOut() {
			fn += ", "
		}
	}
	if t.NumOut() > 1 {
		fn += ")"
	}
	return fn
}

func src2ast(src string) []ast.Decl {
	fmt.Println(src)
	dcls, e := parser.ParseDeclList(token.NewFileSet(), "", src)
	if e != nil {
		panic(e)
	}
	return dcls
}

func remotize0(source string) string {
	decls := src2ast(source)
	fmt.Println(len(decls), "declarations")
	ast.Print(token.NewFileSet(), decls)
	for _, decl := range decls {
		if gendecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gendecl.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if it, ok := ts.Type.(*ast.InterfaceType); ok {
						return remotizeInterface(it)
					}
				}
			}
		}
	}
	return ""
}

func remotizeInterface(iface *ast.InterfaceType) string {
	out := bytes.NewBufferString("")
	for _, f := range iface.Methods.List {
		if ft, ok := f.Type.(*ast.FuncType); ok {
			wrapFunction(out, f.Names[0].Name, ft)
		}
	}
	fmt.Println(out.String())
	return out.String()
}


func wrapFunction(out io.Writer, name string, fun *ast.FuncType) {
	fmt.Fprintf(out, "// wrapper for: %s\n\n", name)
	//argcnt := 
	generateStructWrapper(out, fun.Params, "Args", name)
	//replycnt := 
	generateStructWrapper(out, fun.Results, "Reply", name)
	//generateServerRPCWrapper(out, fun, name, argcnt, replycnt)
	//generateClientRPCWrapper(out, fun, name, argcnt, replycnt)
	fmt.Fprintf(out, "\n")
}

func generateStructWrapper(out io.Writer, fun *ast.FieldList, structname, name string) int {
	fmt.Fprintf(out, "type %s_%s struct {\n", structname, name)
	argn := 0
	if fun == nil {
		return argn
	}
	for _, field := range fun.List {
		fmt.Fprintf(out, "\t")
		// names
		if field.Names != nil {
			for j, _ := range field.Names {
				fmt.Fprintf(out, "Arg%d", argn)
				if j != len(field.Names)-1 {
					fmt.Fprintf(out, ", ")
				}
				argn++
			}
			fmt.Fprintf(out, " ")
		} else {
			fmt.Fprintf(out, "Arg%d ", argn)
			argn++
		}

		// type
		prettyPrintTypeExpr(out, field.Type)

		// \n
		fmt.Fprintf(out, "\n")
	}
	fmt.Fprintf(out, "}\n")
	return argn
}

func prettyPrintTypeExpr(out io.Writer, e ast.Expr) {
	ty := reflect.TypeOf(e)
	switch t := e.(type) {
	case *ast.StarExpr:
		fmt.Fprintf(out, "*")
		prettyPrintTypeExpr(out, t.X)
	case *ast.Ident:
		fmt.Fprintf(out, t.Name)
	case *ast.ArrayType:
		fmt.Fprintf(out, "[]")
		prettyPrintTypeExpr(out, t.Elt)
	case *ast.SelectorExpr:
		prettyPrintTypeExpr(out, t.X)
		fmt.Fprintf(out, ".%s", t.Sel.Name)
	case *ast.FuncType:
		fmt.Fprintf(out, "func(")
		prettyPrintFuncFieldList(out, t.Params)
		fmt.Fprintf(out, ")")

		buf := bytes.NewBuffer(make([]byte, 0, 256))
		nresults := prettyPrintFuncFieldList(buf, t.Results)
		if nresults > 0 {
			results := buf.String()
			if strings.Index(results, " ") != -1 {
				results = "(" + results + ")"
			}
			fmt.Fprintf(out, " %s", results)
		}
	case *ast.MapType:
		fmt.Fprintf(out, "map[")
		prettyPrintTypeExpr(out, t.Key)
		fmt.Fprintf(out, "]")
		prettyPrintTypeExpr(out, t.Value)
	case *ast.InterfaceType:
		fmt.Fprintf(out, "interface{}")
	case *ast.Ellipsis:
		fmt.Fprintf(out, "...")
		prettyPrintTypeExpr(out, t.Elt)
	default:
		fmt.Fprintf(out, "\n[!!] unknown type: %s\n", ty.String())
	}
}

func prettyPrintFuncFieldList(out io.Writer, f *ast.FieldList) int {
	count := 0
	if f == nil {
		return count
	}
	for i, field := range f.List {
		// names
		if field.Names != nil {
			for j, name := range field.Names {
				fmt.Fprintf(out, "%s", name.Name)
				if j != len(field.Names)-1 {
					fmt.Fprintf(out, ", ")
				}
				count++
			}
			fmt.Fprintf(out, " ")
		} else {
			count++
		}

		// type
		prettyPrintTypeExpr(out, field.Type)

		// ,
		if i != len(f.List)-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	return count
}

