// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't hrave to rewrite an interface by
// hand in order to use it remotely or out-of-process. 
package remotize

import (
	"fmt"
	"io"
	"os"
	"rpc"
	"reflect"
	"strings"
	"sync"
)

// Remotized Registry
var registry = make(map[string]interface{})

// Registry's lock
var lock sync.RWMutex

// Remotes (server wrappers) must be Implementers
type BuildService func(*rpc.Server, interface{}) interface{}

// Locals (client wrappers) must be Invokers
type BuildRemote func(*rpc.Client) interface{}

// Register will register a
func Register(r interface{}, br BuildRemote, s interface{}, bs BuildService) {
	cname := fmt.Sprintf("%v", reflect.TypeOf(r))
	sname := fmt.Sprintf("%v", reflect.TypeOf(s))
	lock.Lock()
	defer lock.Unlock()
	registry[cname] = br
	registry[sname] = bs
}

// registryFind will find a registered name in the remotize registry
func registryFind(name string) interface{} {
	lock.Lock()
	defer lock.Unlock()
	return registry[name]
}

func Please(i interface{}) {
	// Nothing to do, just a marker
}

// New Remote Instance by Interface
func NewService(s *rpc.Server, ifaceimpl interface{}) interface{} {
	return NewServiceWith(s, ifaceimpl, ifaceimpl)
}

// New Service With an interface and a diferent implementation
func NewServiceWith(s *rpc.Server, iface interface{},
impl interface{}) interface{} {
	p := registryFind(searchName("", nameFor(iface)) + "Service")
	if p == nil {
		return nil
	}
	return p.(BuildService)(s, impl)
}

// New Local Instance by Interface
func NewRemote(c *rpc.Client, iface interface{}) interface{} {
	p := registryFind(searchName("Remote", nameFor(iface)))
	if p == nil {
		return nil
	}
	return p.(BuildRemote)(c)
}

// Suffix will return the proper "r" or "er" or "" ending as an interface
func Suffix(name string) string {
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

// nameFor returns the name of the given underliying type. Pointers are followed
// up to the final referenced type
func nameFor(i interface{}) string {
	t := reflect.TypeOf(i)
	for t.Kind() == reflect.Ptr {
		t = (t).Elem()
	}
	return fmt.Sprintf("%v", t)
}

// searchName will search a prefix and name on the registry
func searchName(prefix, ifacename string) string {
	parts := strings.Split(ifacename, ".")
	if len(parts) == 2 {
		p := ""
		if !startsWith(parts[1], prefix) {
			p = prefix
		}
		return parts[0] + "." + p + parts[1] + Suffix(ifacename)
	}
	return ifacename + Suffix(ifacename)
}

// startsWith returns true if str starts with substring s
func startsWith(str, s string) bool {
	return len(str) >= len(s) && str[:len(s)] == s
}

// endsWith returns true if str ends with substring s
func endsWith(str, s string) bool {
	return len(str) >= len(s) && str[len(str)-len(s):] == s
}

// endsWith returns true if str ends with an ASCII vowel (a,e,i,o,u)
func endsWithVowel(str string) bool {
	vowels := []string{"a", "e", "i", "o", "u"}
	for _, v := range vowels {
		if str[len(str)-len(v):] == v {
			return true
		}
	}
	return false
}

// Pipe for local invocations, parent/child process communications
type Pipe struct {
	in  io.ReadCloser
	out io.WriteCloser
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

