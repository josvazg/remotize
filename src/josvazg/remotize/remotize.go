// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't hrave to rewrite an interface by
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
	//"sort"
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
func ptr(name string) interface{} {
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
	clt := ptr(ifacename + "Client")
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
	srv := ptr(ifacename + "Server")
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

// Remotized Registry
var registry = make(map[string]interface{})

// Remotes (server wrappers) must be Implementers
type BuildRemote func(*rpc.Server, interface{}) interface{}

// Locals (client wrappers) must be Invokers
type BuildLocal func(*rpc.Client) interface{}

func startsWith(str, s string) bool {
	return len(str) >= len(s) && str[:len(s)] == s
}

func endsWith(str, s string) bool {
	return len(str) >= len(s) && str[len(str)-len(s):] == s
}

func endsWithVowel(str string) bool {
	vowels := []string{"a", "e", "i", "o", "u"}
	for _, v := range vowels {
		if str[len(str)-len(v):] == v {
			return true
		}
	}
	return false
}

func searchName(prefix, ifacename string) string {
	parts := strings.Split(ifacename, ".", -1)
	if len(parts) == 2 {
		p := ""
		if !startsWith(parts[1], prefix) {
			p = prefix
		}
		return parts[0] + "." + p + parts[1] + suffix(ifacename)
	}
	return ifacename + suffix(ifacename)
}

func RegisterRemotized(l interface{}, bl BuildLocal,
r interface{}, br BuildRemote) {
	cname := fmt.Sprintf("%v", reflect.TypeOf(l))
	sname := fmt.Sprintf("%v", reflect.TypeOf(r))
	lock.Lock()
	defer lock.Unlock()
	registry[cname] = bl
	registry[sname] = br
	fmt.Println("Registry is now", registry)
}

func registryFind(name string) interface{} {
	lock.Lock()
	fmt.Println(name, "in", registry, "?")
	defer lock.Unlock()
	return registry[name]
}

// New Remote Instance by Interface
func NewRemote(s *rpc.Server, ifaceimpl interface{}) interface{} {
	return NewRemoteWith(s, ifaceimpl, ifaceimpl)
}

func NewRemoteWith(s *rpc.Server, iface interface{},
impl interface{}) interface{} {
	p := registryFind(searchName("Remote", nameFor(iface)))
	if p == nil {
		return nil
	}
	return p.(BuildRemote)(s, impl)
}

// New Local Instance by Interface
func NewLocal(c *rpc.Client, iface interface{}) interface{} {
	p := registryFind(searchName("Local", nameFor(iface)))
	if p == nil {
		return nil
	}
	return p.(BuildLocal)(c)
}

func Remotize0(i interface{}) os.Error {
	var t reflect.Type
	if _, ok := i.(reflect.Type); ok {
		t = i.(reflect.Type)
	} else {
		t = reflect.TypeOf(i)
	}
	if t.Kind() == reflect.Interface {
		header, declaration := declare(t)
		body := remotize0(header + declaration)
		save("remotized"+t.Name()+".go", header+body)
		return nil
	} else if t.NumMethod() > 0 {
		header, decl := declare(t)
		body := remotize0(header + decl)
		st := t
		for ; st.Kind() == reflect.Ptr; st = st.Elem() {
		}
		save("remotized"+st.Name()+".go", header+decl+body)
		return nil
	}
	if t.Kind() == reflect.Ptr {
		return Remotize0(t.Elem())
	}
	// TODO error
	return nil
}

func save(filename, contents string) {
	f, e := os.Create(filename)
	if e != nil {
		panic(e)
	}
	f.Write(([]byte)(contents))
	f.Close()
}

type declaration struct {
	t       reflect.Type
	src     *bytes.Buffer
	imports map[string]string
}

func suffix(name string) string {
	s := ""
	if !endsWith(name, "er") {
		if endsWithVowel(name) {
			s = "r"
		} else {
			s = "er"
		}
	}
	return s
}

func declare(t reflect.Type) (header string, decl string) {
	var dcl *declaration
	switch t.Kind() {
	case reflect.Interface:
		dcl = newDeclaration(t, "type "+t.Name()+" interface")
	default:
		st := t
		for ; st.Kind() == reflect.Ptr; st = t.Elem() {
		}
		dcl = newDeclaration(t, "type "+st.Name()+suffix(st.Name())+
			" interface")
	}
	dcl.methods(t)
	return dcl.header(), dcl.src.String()
}

func newDeclaration(t reflect.Type, header string) *declaration {
	return &declaration{t, bytes.NewBufferString(header),
		make(map[string]string)}
}

func (d *declaration) methods(t reflect.Type) {
	if t.NumMethod() > 0 {
		fmt.Fprintf(d.src, " {")
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			fmt.Fprintf(d.src, "\n    ")
			d.funcsource(t, &m)
		}
		fmt.Fprintf(d.src, "\n}\n\n")
	}
}

func (d *declaration) funcsource(t reflect.Type, m *reflect.Method) {
	start := 0
	if t.Kind() == reflect.Interface {
		fmt.Fprintf(d.src, m.Name+"(")
	} else if m != nil && m.Name != "" {
		start++
		fmt.Fprintf(d.src, m.Name+"(")
	} else {
		fmt.Fprintf(d.src, "func (")
	}
	if m != nil {
		t = m.Type
	}
	for i := start; i < t.NumIn(); i++ {
		d.source(t.In(i))
		if (i + 1) != t.NumIn() {
			fmt.Fprintf(d.src, ", ")
		}
	}
	fmt.Fprintf(d.src, ") ")
	if t.NumOut() > 1 {
		fmt.Fprintf(d.src, "(")
	}
	for i := 0; i < t.NumOut(); i++ {
		d.source(t.Out(i))
		if (i + 1) != t.NumOut() {
			fmt.Fprintf(d.src, ", ")
		}
	}
	if t.NumOut() > 1 {
		fmt.Fprintf(d.src, ")")
	}
}

func (d *declaration) source(t reflect.Type) {
	switch t.Kind() {
	case reflect.Array:
		fmt.Fprintf(d.src, "["+strconv.Itoa(t.Len())+"]"+t.Name())
	case reflect.Chan:
		fmt.Fprintf(d.src, "chan ")
		d.source(t.Elem())
	case reflect.Func:
		d.funcsource(t, nil)
	case reflect.Map:
		fmt.Fprintf(d.src, "map[")
		d.source(t.Key())
		fmt.Fprintf(d.src, "]")
		d.source(t.Elem())
	case reflect.Ptr:
		fmt.Fprintf(d.src, "*")
		d.source(t.Elem())
	case reflect.Slice:
		fmt.Fprintf(d.src, "[]")
		d.source(t.Elem())
	case reflect.String:
		fmt.Fprintf(d.src, "string")
	default:
		d.pack(t)
		fmt.Fprintf(d.src, t.String())
		return
	}
	d.methods(t)
}

func (d *declaration) pack(t reflect.Type) {
	packpath := t.PkgPath()
	if packpath != "" {
		alias := t.String()[0 : len(t.String())-len(t.Name())-1]
		d.imports[alias] = packpath
	}
}

func packname(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return packname(t.Elem())
	}
	path := t.PkgPath()
	parts := strings.Split(path, ".", -1)
	if parts == nil && len(parts) == 1 {
		return path
	}
	return parts[len(parts)-1]
}

func (d *declaration) header() string {
	packname := packname(d.t)
	buf := bytes.NewBufferString("package " + packname + "\n\n")
	d.imports["rpc"] = "rpc"
	d.imports["os"] = "os"
	if packname != "remotize" {
		d.imports["remotize"] = "remotize"
	}
	if len(d.imports) > 0 {
		fmt.Fprintf(buf, "import (\n")
		for _, i := range d.imports {
			v := d.imports[i]
			if i == v {
				fmt.Fprintf(buf, "    \"%v\"\n", i)
			} else {
				fmt.Fprintf(buf, "    %v \"%v\"\n", v, i)
			}
		}
		fmt.Fprintf(buf, ")\n\n")
	}
	return buf.String()
}

func src2ast(src string) *ast.File {
	//fmt.Println(src)
	f, e := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if e != nil {
		panic(e)
	}
	return f
}

func remotize0(source string) string {
	f := src2ast(source)
	//ast.Print(token.NewFileSet(), f)
	rprefix := "remotize."
	if f.Name.Name == "remotize" {
		rprefix = ""
	}
	for _, decl := range f.Decls {
		if gendecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gendecl.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if it, ok := ts.Type.(*ast.InterfaceType); ok {
						return remotizeInterface(rprefix, ts.Name.Name, it)
					}
				}
			}
		}
	}
	return ""
}

func remotizeInterface(rprefix, ifacename string,
iface *ast.InterfaceType) string {
	out := bytes.NewBufferString("")
	fmt.Fprintf(out, "// Autoregistry\n")
	fmt.Fprintf(out, "func init() {\n")
	fmt.Fprintf(out, "    %sRegisterRemotized(Local%s{},\n",
		rprefix, ifacename)
	fmt.Fprintf(out, "        func(cli *rpc.Client) interface{} "+
		"{ return NewLocal%s(cli) },\n", ifacename)
	fmt.Fprintf(out, "        Remote%s{},\n", ifacename)
	fmt.Fprintf(out, "        func(srv *rpc.Server, i interface{}) "+
		" interface{} { return NewRemote%s(srv,i.(%s)) },\n",
		ifacename, ifacename)
	fmt.Fprintf(out, "    )\n")
	fmt.Fprintf(out, "}\n\n")
	remoteInit(out, ifacename)
	localInit(out, ifacename)
	for _, f := range iface.Methods.List {
		if ft, ok := f.Type.(*ast.FuncType); ok {
			wrapFunction(out, ifacename, f.Names[0].Name, ft)
		}
	}
	//fmt.Println(out.String())
	return out.String()
}

func remoteInit(out io.Writer, ifacename string) {
	fmt.Fprintf(out, "// Remote rpc server wrapper for %s\n", ifacename)
	fmt.Fprintf(out, "type Remote%s struct {\n", ifacename)
	fmt.Fprintf(out, "    srv %s\n", ifacename)
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "// Direct Remote%s constructor\n", ifacename)
	fmt.Fprintf(out, "func NewRemote%s(srv *rpc.Server, impl %s) *Remote%s {\n",
		ifacename, ifacename, ifacename)
	fmt.Fprintf(out, "    r:=&Remote%s{impl}\n", ifacename)
	fmt.Fprintf(out, "    srv.Register(r)\n")
	fmt.Fprintf(out, "    return r\n")
	fmt.Fprintf(out, "}\n\n")
}

func localInit(out io.Writer, ifacename string) {
	fmt.Fprintf(out, "// Local rpc client for %s\n", ifacename)
	fmt.Fprintf(out, "type Local%s struct {\n", ifacename)
	fmt.Fprintf(out, "    cli *rpc.Client\n")
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "// Direct Local%s constructor\n", ifacename)
	fmt.Fprintf(out, "func NewLocal%s(cli *rpc.Client) *Local%s {\n",
		ifacename, ifacename)
	fmt.Fprintf(out, "    return &Local%s{cli}\n", ifacename)
	fmt.Fprintf(out, "}\n\n")
}

func wrapFunction(out io.Writer, iface, name string, fun *ast.FuncType) {
	fmt.Fprintf(out, "// wrapper for: %s\n\n", name)
	argcnt := generateStructWrapper(out, fun.Params, "Args", name)
	results, inouts := prepareInOuts(fun.Params, fun.Results)
	replycnt := generateStructWrapper(out, results, "Reply", name)
	generateServerRPCWrapper(out, fun, iface, name, argcnt, replycnt, inouts)
	generateClientRPCWrapper(out, fun, iface, name, argcnt, replycnt, inouts)
	fmt.Fprintf(out, "\n")
}

func prepareInOuts(params *ast.FieldList, r *ast.FieldList) (*ast.FieldList,
[]int) {
	args := make([]*ast.Field, 0)
	inouts := make([]int, 0)
	for n, field := range params.List {
		if _, ok := field.Type.(*ast.StarExpr); ok {
			args = append(args, field)
			inouts = append(inouts, n)
		}
	}
	if r == nil {
		r = &ast.FieldList{List: args}
	} else {
		r.List = append(r.List, args...)
	}
	return r, inouts
}

func generateStructWrapper(out io.Writer, fun *ast.FieldList, structname,
name string) int {
	fmt.Fprintf(out, "type %s_%s struct {\n", structname, name)
	defer fmt.Fprintf(out, "}\n\n")
	argn := 0
	if fun == nil || len(fun.List) == 0 {
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

// function that is being exposed to an RPC API, but calls simple "Server_" one
func generateServerRPCWrapper(out io.Writer, fun *ast.FuncType,
iface, name string, argcnt, replycnt int, inouts []int) {
	fmt.Fprintf(out, "func (r *Remote%s) %s(args *Args_%s, "+
		"reply *Reply_%s) os.Error {\n", iface, name, name, name)

	fmt.Fprintf(out, "\t")
	replies := replycnt - len(inouts)
	for i := 0; i < replies; i++ {
		fmt.Fprintf(out, "reply.Arg%d", i)
		if i != replycnt-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	if replies > 0 {
		fmt.Fprintf(out, " = ")
	}
	fmt.Fprintf(out, "r.srv.%s(", name)
	for i := 0; i < argcnt; i++ {
		fmt.Fprintf(out, "args.Arg%d", i)
		if i != argcnt-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	fmt.Fprintf(out, ")\n")
	for i := replies; i < replycnt; i++ {
		fmt.Fprintf(out, "\treply.Arg%d=args.Arg%d\n", i, inouts[i-replies])
	}
	fmt.Fprintf(out, "\treturn nil\n}\n\n")
}

func generateClientRPCWrapper(out io.Writer, fun *ast.FuncType, iface,
name string, argcnt, replycnt int, inouts []int) {
	fmt.Fprintf(out, "func (l *Local%s) %s(", iface, name)
	prettyPrintFuncFieldListUsingArgs(out, fun.Params)
	fmt.Fprintf(out, ")")

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	nresults := prettyPrintFuncFieldList(buf, fun.Results)
	if nresults > 0 {
		results := buf.String()
		if strings.Index(results, " ") != -1 {
			results = "(" + results + ")"
		}
		fmt.Fprintf(out, " %s", results)
	}
	fmt.Fprintf(out, " {\n")
	fmt.Fprintf(out, "\tvar args Args_%s\n", name)
	fmt.Fprintf(out, "\tvar reply Reply_%s\n", name)
	for i := 0; i < argcnt; i++ {
		fmt.Fprintf(out, "\targs.Arg%d = Arg%d\n", i, i)
	}
	fmt.Fprintf(out, "\terr := l.cli.Call(\"Remote%s.%s\", &args, &reply)\n",
		iface, name)
	fmt.Fprintf(out, "\tif err != nil {\n")
	fmt.Fprintf(out, "\t\tpanic(err.String())\n\t}\n")

	replies := replycnt - len(inouts)
	for i := replies; i < replycnt; i++ {
		fmt.Fprintf(out, "\t*Arg%d=*reply.Arg%d\n", i, inouts[i-replies])
	}
	fmt.Fprintf(out, "\treturn ")
	for i := 0; i < replies; i++ {
		fmt.Fprintf(out, "reply.Arg%d", i)
		if i != replycnt-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	fmt.Fprintf(out, "\n}\n\n")
}

func prettyPrintFuncFieldListUsingArgs(out io.Writer, f *ast.FieldList) int {
	count := 0
	if f == nil {
		return count
	}
	for i, field := range f.List {
		// names
		if field.Names != nil {
			for j, _ := range field.Names {
				fmt.Fprintf(out, "Arg%d", count)
				if j != len(field.Names)-1 {
					fmt.Fprintf(out, ", ")
				}
				count++
			}
			fmt.Fprintf(out, " ")
		} else {
			fmt.Fprintf(out, "Arg%d ", count)
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

