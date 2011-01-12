// Copyright 2011 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style
package rii

import (
	"os"
	"io"
	"gob"
	"rpc"
	"sync"
)

// Invoker interface to satify by any rii-like remotizing transport
//
// Invoke() function will call the function number (fn) with arguments (args) 
// and get their results (r) or an error (e)
type Invoker interface {
	Invoke(fn int, args ...interface{}) ([]interface{}, os.Error)
}

// Invoker using the rpc package as rii transport
type rpcInvoker struct {
	client		*rpc.Client
	funcNames	[]string
}

// Invoker using a local pipe to a local process as rii transport
type pipeInvoker struct {
	pipe		io.ReadWriteCloser		// Data Transport Pipe
	e       	*gob.Encoder			// Encoder to send gobs to the skel
	d       	*gob.Decoder			// Decoder to receive gobs from the skel
	mutex		sync.Mutex				// Mutex for sharing the pipe transport
}

// Pipe Invoker call message
type pipeCall struct {
	fn		int
	args	[]interface{}
}

// Pipe Invoker response message
type pipeResponse struct {
	results	[]interface{}
	error	os.Error
}

// Create a new Rpc Invoker
func NewRpcInvoker(client *rpc.Client, funcNames []string) *rpcInvoker {
	return &rpcInvoker{client,funcNames}
}

// Create a new Pipe Invoker
func NewPipeInvoker(pipe io.ReadWriteCloser) *pipeInvoker {		
	return &pipeInvoker{pipe: pipe,
		e: gob.NewEncoder(pipe),
		d: gob.NewDecoder(pipe)}
}

// Call to a remote (rpc) method
// No need to synchronize the transport here, rpc does it already
func (r *rpcInvoker) Invoke(fn int, 
		args ...interface{}) ([]interface{}, os.Error) {
	var rsp []interface{}
	err:=r.client.Call(r.funcNames[fn],args,rsp)
	return rsp,err
}

// Call to a local (piped) method
func (p *pipeInvoker) Invoke(fn int, 
		args ...interface{}) ([]interface{}, os.Error) {
	call:=&pipeCall{fn,args}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	err:=p.e.Encode(call)
	if(err!=nil) {
		return nil,err
	}
	var rsp pipeResponse
	err=p.d.Decode(&rsp)
	if(err!=nil) {
		return nil,err
	}
	return rsp.results,rsp.error
}



