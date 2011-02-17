// Copyright 2011 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style
package rii

import (
	"fmt"
	"io"
	"os"
	"rpc"
	"time"
)

// Error handler interface
type ErrorHandling func(string, os.Error)

// RIIClient using the rpc package as rii transport
type Client struct {
	client		*rpc.Client		// rpc transport
	handler		ErrorHandling	// default error handler
	timeout		int64			// default rpc max timeout
}

// UNSET_TIMEOUT
const NoTimeout=0

// Rii default error handling routine for all remote interfaces
var DefaultErrorHandling	ErrorHandling

// Create a new RII Client
func NewClient(rwc io.ReadWriteCloser) *Client {
	return &Client{rpc.NewClient(rwc),nil,NoTimeout}
}

// RII Call to a method
// No need to synchronize the transport here, rpc does it already
func (c *Client) Call(method string, args...interface{}) (*[]interface{}, 
		os.Error) {
	var res []interface{}
	var e os.Error
	if(c.timeout==NoTimeout) {
		e=c.client.Call(method,args,&res)
	} else {
		e=c.callTimeout(method,args,&res,c.timeout)
    }
	return &res,e
}

// Call with timeout
func (c *Client) callTimeout(method string, args []interface{}, 
	reply *[]interface{}, timeout int64) os.Error { 
	call := c.client.Go(method, args, reply,nil) 
	select { 
	case <-call.Done: 
			// Call returned
	case <-time.After(timeout): 
			msg:=fmt.Sprintf("Call timed out %vms at %v()!",timeout,method)
			return os.NewError(msg)
    }
	return call.Error
}

// Handle an error
func (c *Client) HandleError(funcname string, e os.Error) {
	if(c.handler!=nil) {
		c.handler(funcname,e)
	} else if(DefaultErrorHandling!=nil) {
		DefaultErrorHandling(funcname,e)
	} else {
		errmsg:=fmt.Sprintf("Error at %v(): %v",funcname,e)
		panic(errmsg)
	}
}

// Set remote interface error handler
func (c *Client) ErrorHandler(f ErrorHandling) {
	c.handler=f
}

// Set remote interface default timeout
func (c* Client) Timeout(timeout int64) {
	c.timeout=timeout
}

// Pipe for local invocations, parent/child comms
type pipe struct {
	in io.ReadCloser
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
	err:=p.in.Close()
	if(err!=nil) {
		return err
	}
	return p.out.Close()
}

// Prepare a ReadWriteCloser pipe from a reader and a writer
// This can be passed to NewClient to use RPCs over local pipe streams
func IO(in io.ReadCloser, out io.WriteCloser) *pipe {
	return &pipe{in,out}
}

