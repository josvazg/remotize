// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't have to rewrite an interface by
// hand in order to use it remotely or out-of-process. 
package remotize

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"http"
	"io"
	"net"
	"os"
	"reflect"
	"rpc"
	"sort"
	"strings"
	"sync"
	"template"
	"time"
)

// Setter interface to initialize the Servers
type Setter interface {
	set(interface{})
}

// Remotized Registry
type Registry struct {
	remotized map[string]reflect.Type
	lock      sync.RWMutex
}

// Remotized types registry
var Remotized = &Registry{remotized: make(map[string]reflect.Type)}

// Add to remotized registry
func (r *Registry) Add(t reflect.Type) {
	name := fmt.Sprintf("%v", t)
	r.lock.Lock()
	r.remotized[name] = t
	fmt.Println("Registry is now", r.remotized)
	r.lock.Unlock()
}

// Remove from registry
func (r *Registry) Remove(name string) {
	r.lock.Lock()
	r.remotized[name] = nil, false
	r.lock.Unlock()
}

// Find from registry
func (r *Registry) find(name string) reflect.Type {
	r.lock.RLock()
	defer r.lock.RLock()
	return r.remotized[name]
}

// Contains replies to 'Is this name contained in the registry?'
func (r *Registry) contains(name string) bool {
	r.lock.RLock()
	defer r.lock.RLock()
	_, ok := r.remotized[name]
	return ok
}

// Instatiante returns an instance of the given type if found in the registry, 
// or nil otherwise
func (r *Registry) instantiate(name string) interface{} {
	t := r.find(name)
	if t == nil {
		return nil
	}
	return reflect.MakeZero(t)
}

// named returns the name of the given underliying type. Pointers are followed
// up to the final referenced type
func nameFor(i interface{}) string {
	t := reflect.Typeof(i)
	for t.Kind() == reflect.Ptr {
		t = (t).(*reflect.PtrType).Elem()
	}
	return fmt.Sprintf("%v", t)
}

// NewClient instantiates a client for the given interface name, 
// if found on the registry, otherwise nil is returned.
// If pack is NOT empty that package name is used to locate the remotization,
// use it when the remotized code is on a different package from the 
// remotized interface
func (r *Registry) NewClient(ifacename, pack string) interface{} {
	if pack != "" {
		dotpos := strings.LastIndex(ifacename, ".")
		ifacename = pack + "." + ifacename[dotpos:]
	}
	return r.instantiate(ifacename + "Client")
}

// NewClientFor instantiates (if found) the client for the given interface,
// otherwise nil is returned
func (r *Registry) NewClientFor(i interface{}, pack string) interface{} {
	return r.NewClient(nameFor(i), pack)
}

// NewServer instantiates a server for the given interface name,
// if found on the registry, otherwise nil is returned.
// It also initiates it with the implementation of that interface
func (r *Registry) NewServer(ifacename, pack string, impl interface{}) interface{} {
	if pack != "" {
		dotpos := strings.LastIndex(ifacename, ".")
		ifacename = pack + "." + ifacename[dotpos:]
	}
	s := r.instantiate(ifacename + "Server")
	if s != nil {
		(s).(Setter).set(impl)
	}
	return s
}

// NewServerFor instantiates a server for the given interface,
// if found on the registry, otherwise nil is returned.
// It also initiates it with the implementation of that interface
func (r *Registry) NewServerFor(i, impl interface{}, pack string) interface{} {
	return r.NewServer(nameFor(i), pack, impl)
}

// Error handler interface
type ErrorHandling func(string, os.Error)

// RemotizeClient using the rpc package as transport
type Client struct {
	client  *rpc.Client   // rpc transport
	handler ErrorHandling // default error handler
	timeout int64         // default rpc max timeout
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
type Timeout struct {
	os.Error
	TimeoutCall *rpc.Call
}

// UNSET_TIMEOUT
const NoTimeout = 0

// Default Remotize error handling routine for all remote interfaces
var DefaultErrorHandling ErrorHandling

// Create a new Remotized Client
func NewClient(rwc io.ReadWriteCloser) *Client {
	return &Client{rpc.NewClient(rwc), nil, NoTimeout}
}

// Remotize Call to a method
// No need to synchronize the transport here, rpc does it already
func (c *Client) Call(method string, args ...interface{}) (*Results, os.Error) {
	var r Results
	a := &Args{args}
	var e os.Error
	if c.timeout == NoTimeout {
		e = c.client.Call(method, a, &r)
	} else {
		e = c.callTimeout(method, a, &r, c.timeout)
	}
	return &r, e
}

// Call with timeout
func (c *Client) callTimeout(method string, args interface{},
reply interface{}, timeout int64) os.Error {
	call := c.client.Go(method, args, reply, nil)
	select {
	case <-call.Done:
		// Call returned
	case <-time.After(timeout):
		msg := fmt.Sprintf("Call timed out %vms at %v()!", timeout, method)
		return &Timeout{os.NewError(msg), call}
	}
	return call.Error
}

// Handle an error
func (c *Client) HandleError(funcname string, e os.Error) {
	if c.handler != nil {
		c.handler(funcname, e)
	} else if DefaultErrorHandling != nil {
		DefaultErrorHandling(funcname, e)
	} else {
		errmsg := fmt.Sprintf("Error at %v(): %v", funcname, e)
		panic(errmsg)
	}
}

// Set remote interface error handler
func (c *Client) ErrorHandler(f ErrorHandling) {
	c.handler = f
}

// Set remote interface default timeout
func (c *Client) Timeout(timeout int64) {
	c.timeout = timeout
}

// Pipe for local invocations, parent/child comms
type pipe struct {
	in  io.ReadCloser
	out io.WriteCloser
}

// Read from the pipe
func (p *pipe) Read(b []byte) (n int, err os.Error) {
	return p.in.Read(b)
}

// Write to the pipe
func (p *pipe) Write(b []byte) (n int, err os.Error) {
	return p.out.Write(b)
}

// Close pipe io
func (p *pipe) Close() os.Error {
	err := p.in.Close()
	if err != nil {
		return err
	}
	return p.out.Close()
}

// Prepare a ReadWriteCloser pipe from a reader and a writer
// This can be passed to NewClient to use RPCs over local pipe streams
func IO(in io.ReadCloser, out io.WriteCloser) *pipe {
	return &pipe{in, out}
}

// Remotized Server using the rpc package as transport
type Server struct {
	server *rpc.Server // rpc server
	Iface  interface{} // iface to be invoked
}

// Create a new Remotized Server on a local io pipe
func NewServer(iface interface{}) *Server {
	s := &Server{rpc.NewServer(), iface}
	s.server.Register(iface)
	return s
}

// Do serve on a (local) pipe
func (s *Server) ServePipe(rwc io.ReadWriteCloser) {
	go s.server.ServeConn(rwc)
}

// Do serve on a network listener
func (s *Server) Serve(l net.Listener) {
	http.Serve(l, nil)
}


const wrapsrc = `// Autogenerated Remotize Interface ${Iface} wrapper [DO NOT EDIT!]
package ${Pack}

import (
${Imports}
)

// Autoregistry
func init() {
	${Prefix}Remotized.Add(reflect.Typeof(${Iface}Server{}))
	${Prefix}Remotized.Add(reflect.Typeof(${Iface}Client{}))
}

// Server wrapper for ${Iface}
type ${Iface}Server struct {
	s	${Iface}
}

// Client wrapper for ${Iface}
type ${Iface}Client struct {
	c	*${Prefix}Client
}

// Setting the implementation
func (s *${Iface}Server) set(s interface{}) {
	s.s=s
}

${Calls}
`

type methodInfo struct {
	m    reflect.Method
	re   bool
	pos  int
	ptrs []int
}

type wrapgen struct {
	Iface   string
	Pack    string
	Prefix  string
	Imports *bytes.Buffer
	Calls   *bytes.Buffer
	imports []string
	methods []methodInfo
}

// Remotize will create the rpc client/server file needed to use some given 
// interface remotely
func Remotize(iface interface{}) {
	if it, ok := iface.(*reflect.InterfaceType); ok {
		remotize(it, "")
		return
	}
	t := reflect.Typeof(iface)
	if pt, ok := t.(*reflect.PtrType); ok {
		if it, ok2 := pt.Elem().(*reflect.InterfaceType); ok2 {
			remotize(it, "")
			return
		}
	}
	fmt.Println("Can't remotize", iface, "of non interface type", t)
}

// remotize will remotize the interface by generating a proper rpc client/server
// wrapping
func remotize(it *reflect.InterfaceType, pack string) {
	fmt.Println("Remotizing interface", it)
	if pack == "" {
		pack = it.PkgPath()
	}
	w := newWrapgen(it.Name(), pack)

	nm := it.NumMethod()
	fmt.Println("Interface exports ", nm, "methods")
	for i := 0; i < nm; i++ {
		m := it.Method(i)
		w.addMethod(m)
	}
	w.genWrapper(it)
}

// newWrapgen creates an interface wrapper generator
func newWrapgen(Ifacename, pack string) *wrapgen {
	w := &wrapgen{Iface: Ifacename,
		Pack:    pack,
		Calls:   bytes.NewBuffer(make([]byte, 0)),
		imports: []string{"os", "reflect"},
	}
	if pack != "remotize" {
		w.imports = append(w.imports, "remotize")
		w.Prefix = "remotize."
	} else {
		w.Prefix = ""
	}
	return w
}

// addMethod wraps another method fo the interface
func (w *wrapgen) addMethod(m reflect.Method) {
	re, pos := returnsError(m)
	ptrs := inouts(m)
	nin := m.Type.NumIn()
	for i := 0; i < nin; i++ {
		w.addImport(m.Type.In(i).PkgPath())
	}
	nout := m.Type.NumOut()
	for i := 0; i < nout; i++ {
		w.addImport(m.Type.Out(i).PkgPath())
	}
	w.methods = append(w.methods, methodInfo{m, re, pos, ptrs})
	w.clientWrapper()
	w.serverWrapper()
}

// addImport adds an import if needed
func (w *wrapgen) addImport(imp string) {
	if imp == "" { // empty should not be imported
		return
	}
	for _, i := range w.imports {
		if i == imp { // already imported
			return
		}
	}
	w.imports = append(w.imports, imp)
}

// genWrapper generates the final source code for the wrapped interface
func (w *wrapgen) genWrapper(it *reflect.InterfaceType) {
	sort.SortStrings(w.imports)
	w.Imports = bytes.NewBuffer(make([]byte, 0))
	for _, s := range w.imports {
		fmt.Fprintf(w.Imports, "\"%v\"\n", s)
	}
	src := bytes.NewBuffer(make([]byte, 0))
	t := template.New(nil)
	t.SetDelims("${", "}")
	e := t.Parse(wrapsrc)
	if e != nil {
		fmt.Println("Error:", e)
	}
	t.Execute(src, w)
	fset := token.NewFileSet()
	filename := strings.ToLower(it.Name()) + "Remotized.go"
	f, e := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if e != nil {
		fmt.Println("Error:", e)
		fmt.Print("src:\n", src)
	}
	fos, e := os.Open(filename, os.O_CREATE|os.O_WRONLY, 0755)
	if e != nil {
		fmt.Println("Error:", e)
	}
	pcfg := &printer.Config{printer.TabIndent, 4}
	pcfg.Fprint(fos, fset, f)
	fos.Close()
}

// clientWrapper genrates the whole client wrapping method
func (w *wrapgen) clientWrapper() {
	mi := w.methods[len(w.methods)-1]
	w.methodSignature(mi.m)
	fmt.Fprintf(w.Calls, " {\n")
	w.wrapCall(mi)
	w.clientReturn(mi)
	fmt.Fprintf(w.Calls, "}\n\n")
}

// wrapCall wrapps the call to the server RPC
func (w *wrapgen) wrapCall(mi methodInfo) {
	m := mi.m
	r := "r"
	if m.Type.NumOut()+len(mi.ptrs) == 0 {
		r = "_"
	}
	fmt.Fprintf(w.Calls, "\t%v, e := c.c.Call(\"%vServer.RPC%v\",", r, w.Iface,
		m.Name)
	nin := m.Type.NumIn()
	for i := 0; i < nin; i++ {
		if i > 0 {
			fmt.Fprintf(w.Calls, ",")
		}
		fmt.Fprintf(w.Calls, " a%v", (i + 1))
	}
	fmt.Fprintf(w.Calls, ")\n")
}

// methodSignature generates the client wrapper method signature
func (w *wrapgen) methodSignature(m reflect.Method) {
	fmt.Fprintf(w.Calls, "// %v.%v Client wrapper\n", w.Iface, m.Name)
	fmt.Fprintf(w.Calls, "func (c *%vClient) %v(", w.Iface, m.Name)
	nin := m.Type.NumIn()
	for i := 0; i < nin; i++ {
		if i > 0 {
			fmt.Fprintf(w.Calls, ",")
		}
		fmt.Fprintf(w.Calls, "a%v %v", (i + 1), m.Type.In(i).String())
	}
	fmt.Fprintf(w.Calls, ")")
	nout := m.Type.NumOut()
	if nout > 0 {
		fmt.Fprintf(w.Calls, " ")
		if nout > 1 {
			fmt.Fprintf(w.Calls, "(")
		}
		for i := 0; i < nout; i++ {
			if i > 0 {
				fmt.Fprintf(w.Calls, ",")
			}
			fmt.Fprintf(w.Calls, "%v", m.Type.Out(i))
		}
		if nout > 1 {
			fmt.Fprintf(w.Calls, ") ")
		}
	}
}

// clientReturn generates the client wrapper return, including error handling
// if needed
func (w *wrapgen) clientReturn(mi methodInfo) {
	m := mi.m
	if !mi.re {
		fmt.Fprintf(w.Calls, "\tif e != nil {\n")
		fmt.Fprintf(w.Calls, "\t\tc.c.HandleError(\"%v.%v\", e)\n", w.Iface,
			m.Name)
		fmt.Fprintf(w.Calls, "\t}\n")
	}
	nout := m.Type.NumOut()
	ninouts := len(mi.ptrs)
	for i := 0; i < ninouts; i++ {
		fmt.Fprintf(w.Calls, "\t*a%v=(r.R[%v]).(%v)\n", mi.ptrs[i]+1, nout+i,
			m.Type.In(mi.ptrs[i]).(*reflect.PtrType).Elem())
	}
	if nout > 0 {
		fmt.Fprintf(w.Calls, "\treturn ")
		for i := 0; i < nout; i++ {
			if i != 0 {
				fmt.Fprintf(w.Calls, ", ")
			}
			if i == mi.pos {
				fmt.Fprintf(w.Calls, "e")
			} else {
				fmt.Fprintf(w.Calls, "(r.R[%v]).(%v)", i, m.Type.Out(i))
			}
		}
		fmt.Fprintf(w.Calls, "\n")
	}
}

// serverWrapper generates the server call wrapper
func (w *wrapgen) serverWrapper() {
	mi := w.methods[len(w.methods)-1]
	m := mi.m
	fmt.Fprintf(w.Calls, "// %v.%v Server wrapper\n", w.Iface, m.Name)
	fmt.Fprintf(w.Calls, "func (s *%vServer) RPC%v(", w.Iface, m.Name)
	fmt.Fprintf(w.Calls, "a *Args, r *Results) os.Error {\n")
	nout := m.Type.NumOut()
	ninouts := len(mi.ptrs)
	if nout+ninouts > 0 {
		fmt.Fprintf(w.Calls, "\tr.R= make([]interface{}, %v)\n", nout+ninouts)
	}
	for i := 0; i < ninouts; i++ {
		fmt.Fprintf(w.Calls, "\ta%v := (a.A[%v]).(%v)\n", mi.ptrs[i]+1, nout+i,
			m.Type.In(mi.ptrs[i]).(*reflect.PtrType).Elem())
		fmt.Fprintf(w.Calls, "\tr.R[%v] = &a%v\n", nout+i, mi.ptrs[i]+1)
	}
	fmt.Fprintf(w.Calls, "\t")
	for i := 0; i < nout; i++ {
		if i != 0 {
			fmt.Fprintf(w.Calls, ", ")
		}
		fmt.Fprintf(w.Calls, "r.R[%v]", i)
	}
	if nout > 0 {
		fmt.Fprintf(w.Calls, " = ")
	}
	fmt.Fprintf(w.Calls, "s.s.%v(", m.Name)
	nin := m.Type.NumIn()
	j := 0
	for i := 0; i < nin; i++ {
		if i != 0 {
			fmt.Fprintf(w.Calls, ", ")
		}
		if j < ninouts && i == mi.ptrs[j] {
			fmt.Fprintf(w.Calls, "(r.R[%v]).(%v)", nout+j, m.Type.In(mi.ptrs[i]))
			j++
		} else {
			fmt.Fprintf(w.Calls, "(a.A[%v]).(%v)", i, m.Type.In(i))
		}
	}
	fmt.Fprintf(w.Calls, ")\n")
	if mi.re {
		fmt.Fprintf(w.Calls, "\tif r.R[%v] != nil {\n", mi.pos)
		fmt.Fprintf(w.Calls, "\t\treturn (r.R[%v]).(os.Error)\n", mi.pos)
		fmt.Fprintf(w.Calls, "\t}\n")
	}
	fmt.Fprintf(w.Calls, "\treturn nil\n")
	fmt.Fprintf(w.Calls, "}\n\n")
}

// returnsError says whether a method returns an os.Error and where
func returnsError(m reflect.Method) (hasError bool, pos int) {
	errorType := reflect.Typeof((*os.Error)(nil)).(*reflect.PtrType).Elem()
	nout := m.Type.NumOut()
	for i := 0; i < nout; i++ {
		if m.Type.Out(i) == errorType {
			return true, i
		}
	}
	return false, -1
}

// inouts returns an array with the positions (starting at o) of input 
// parameters that are pointers. 
// Those pointers should be treated as input/output parameters
func inouts(m reflect.Method) []int {
	nin := m.Type.NumIn()
	ptrs := make([]int, 0)
	for i := 0; i < nin; i++ {
		if m.Type.In(i).Kind() == reflect.Ptr {
			ptrs = append(ptrs, i)
		}
	}
	return ptrs
}

