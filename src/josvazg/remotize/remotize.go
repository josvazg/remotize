// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// The remotize package wraps rpc calls so you don't have to. 
//
// This root remotize package is the only dependency your programs needs to 
// import within their application. The tool that remotizes you code is on 
// separated packages that won't be linked to your executable unless, of course, 
// you import it especifically for some reason.
// 
package remotize

import (
	"fmt"
	"josvazg/remotize/misc"
	"rpc"
	"reflect"
	"strings"
)

// Please does nothing. It is just a marker that tells the remotize tool 
// (goremote) that i interface must, "please", be remotized:
//   
//  import remotize
//  ...
//  remotize.Please(new(somepackage.UrlStorer))
//
func Please(i interface{}) {
	// Nothing to do, just a marker
}

// NewService returns a new service wrapper to serve calls to 'ifaceimpl' using 
// 's' RpcServer as transport, so that remote instances will be able to call
// ifaceimple methods.
func NewService(s *rpc.Server, ifaceimpl interface{}) interface{} {
	return NewServiceWith(s, ifaceimpl, ifaceimpl)
}

// NewServiceWith returns a new service wrapper to call 'impl', with interface 
// 'iface' using 's' RpcServer.
func NewServiceWith(s *rpc.Server, iface interface{},
impl interface{}) interface{} {
	p := misc.RegistryFind(searchName("", nameFor(iface)) + "Service")
	if p == nil {
		return nil
	}
	return p.(misc.BuildService)(s, impl)
}

// NewRemote returns a local reference to a remote interface of type iface,
// reachable through c RpcClient.
func NewRemote(c *rpc.Client, iface interface{}) interface{} {
	p := misc.RegistryFind(searchName("Remote", nameFor(iface)))
	if p == nil {
		return nil
	}
	return p.(misc.BuildRemote)(c)
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
		if !misc.StartsWith(parts[1], prefix) {
			p = prefix
		}
		return parts[0] + "." + p + parts[1] + misc.Suffix(ifacename)
	}
	return ifacename + misc.Suffix(ifacename)
}

