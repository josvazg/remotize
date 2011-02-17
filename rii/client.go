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

// Caller interface to satify by any rii-like remotizing transport
//
// Call() function will call the function number (fn) with arguments (args) 
// and get their results (r) or an error (e)
type Caller interface {
	Call(string, bool, ...interface{}) ([]interface{}, os.Error)
}

// Error handler interface
type ErrorHandling func(string, os.Error)

// RIIClient using the rpc package as rii transport
type Client struct {
	client		*rpc.Client		// rpc transport
	handler		ErrorHandling	// default error handler
	timeout		*int			// default rpc max timeout
}

// UNSET_TIMEOUT
const NoTimeout=0

// Rii default error handling routine for all remote interfaces
var DefaultErrorHandling	ErrorHandling

// Rii default call timeout (0 is no timeout) for all remote interfaces
var DefaultTimeout			int

// Create a new RII Client
func NewClient(rwc io.ReadWriteCloser) *Client {
	return &Client{rpc.NewClient(rwc),nil,nil}
}

// RII Call to a method
// No need to synchronize the transport here, rpc does it already
func (c *Client) Call(funcname string, re bool, 
		args ...interface{}) ([]interface{}, os.Error) {
	var rsp []interface{}
	timeout:=c.Timeout(funcname)
	var err os.Error
	if(timeout==NoTimeout) {
		err=c.client.Call(funcname,args,rsp)
	} else {
		rch:=make(chan *rpc.Call)
		c.client.Go(funcname,args,rsp,rch)
		timeoutCh := make(chan bool, 1)
		go func() {
		    time.Sleep(int64(timeout)*1e6)
		    timeoutCh <- true
		}()
		select {
			case call:=<-rch:
			    rsp=(call.Reply).([]interface{})
				err=call.Error
			case <-timeoutCh:
				msg:=fmt.Sprintf("Timeout %vms at %v()!",timeout,funcname)
				err=os.NewError(msg)
		} 
	} 
	if(err!=nil && !re) {
		c.handleError(funcname,err)
	}
	return rsp,err
}

// Effective Timeout
func (c *Client) Timeout(funcname string) int {
	if(c.timeout!=nil) {
		return *c.timeout
	}
	return DefaultTimeout
}

// Handle an error
func (c *Client) handleError(funcname string, e os.Error) {
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
func (c* Client) InterfaceTimeout(timeout int) {
	c.timeout=&timeout
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

