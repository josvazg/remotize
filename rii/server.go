// Copyright 2011 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style
package rii

import (
	"io"
	"rpc"
	"net"
	"http"
)

// RII Server using the rpc package as rii transport
type Server struct {
	server		*rpc.Server				// rpc server
	Iface		interface{}				// iface to be invoked
	Service		interface{}				// type implementing the service
}

// Create a new RII Server on a local io pipe
func NewServer(iface interface{}, service interface{}) *Server {
	s:=&Server{rpc.NewServer(),iface,service}
	rpc.Register(service)
	return s
}

// Do serve on a (local) pipe
func (s *Server) ServePipe(rwc io.ReadWriteCloser) {
	s.server.ServeConn(rwc)
}

// Do serve on a network listener
func (s *Server) Serve(l net.Listener) {
	http.Serve(l,nil)
}

