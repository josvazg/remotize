// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't have to rewrite an interface by
// hand in order to use it remotely or out-of-process. 
package remotize

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"rpc"
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

// Add a remotized type to the registry. The type is Exported since that moment
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

// Remove a type from registry. The type is UnExported since that moment
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

// HandleError handles an error
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

