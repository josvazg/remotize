// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

package rii

import (
	"os"
	"io"
	"gob"
	"fmt"
)

// quit: funcNum to close the channel
const (
	quit = -1
)

type Invoker interface {
	Invoke(fn int, args ...interface{}) (r *[]interface{}, e os.Error)
}

// interface for executor implementation's to execute the remotized methods 
type executor interface {
	execute(funcNum int, args []interface{}) []interface{}
}

// invocation message 
type invocation struct {
	id      int	// identifier of the invocation in the asynchronous channel
	funcNum int	// function number to identify the invoked function
	args    []interface{} // input arguments to send to the server skeletor
}

type invocontext struct {
	invocation	// invocation msg
	rch chan *response	// channel to send the final response to
}

type response struct {
	id      int		// identifier of the invocation 
	results []interface{} // output results of the remote executions
	error   os.Error	// error msg (if any, nil otherwise)
}

// stub base type (stub constructor)
type Stub struct {
	rwc     io.ReadWriteCloser		// The conn. or pipe to read/write gobs
	quit    chan int				// Quit channel for closing the stub
	e       *gob.Encoder			// Encoder to send gobs to the skel
	d       *gob.Decoder			// Decoder to receive gobs from the skel
	alive   bool					// 'Stub is alive' flag
	ch      chan *invocontext		// Channel to the invocation sender 
	pending map[int]*invocontext    // Pending invocation contexts
}

// stub base 'constructor' 
func newStub(rwc io.ReadWriteCloser) *Stub {
	st := &Stub{rwc, make(chan int),
		gob.NewEncoder(rwc), gob.NewDecoder(rwc), true,
        make(chan *invocontext), make(map[int]*invocontext)}
	go stubReceiver(st)
	go stubSender(st)
	return st
}

// invocation function to start the remote invocation and receive the result
// The invocation is sent throug a channel to the stubSender() and the reply
// is received from a channel from the stubReceiver() 
func (st *Stub) Invoke(fn int, args ...interface{}) (*[]interface{},os.Error) {
	rch := make(chan *response) // reply channel
	st.ch <- &invocontext{invocation{0, fn, args}, rch} //invoke
	rsp := <-rch // get return
	if rsp.error != nil {
		return nil, rsp.error
	}
	return &rsp.results, nil
}

// close the stub goroutines in an orderly manner 
func (st *Stub) close() {
	if st.alive {
		st.alive = false // the loops are not alive any more
		st.rwc.Close() // the returnReceiver is stopped
		<-st.quit		// wait for the invocationSender to end
	}
}

// stub invocation sender loop goroutine receives invocation requests and sends
// them as gobs over a connection or pipe
func stubSender(st *Stub) {
	id := 0
	for st.alive {
		ictx := <-st.ch
		if ictx.id == quit { // got quit signal
			continue // quit!
		}
		id++ 
		ictx.id = id
		st.pending[ictx.id] = ictx // remember pending invocation context
		err := st.e.Encode(ictx.invocation) // Encode=send invocation
		if err != nil {
			ictx.rch <- &response{id, nil, err} // send/encode error
		}
	}
	fmt.Println("stubSender closed")
	st.quit <- 1
}

// stub response receiver gets gob responses and, after looking up the
// invocontext by id, it replies to the right invoke() goroutine
func stubReceiver(st *Stub) {
	for st.alive {
		var rsp response
		var ictx *invocontext
		err := st.d.Decode(&rsp)
		if err == os.EOF { // EOF?
			if(st.alive) { // got quit signal by colsing the reader
				fmt.Println("warn: stubReceiver remotely stopped!")
				st.alive = false // remote quit
			}
			continue // quit
		}
		if err == nil {
			ictx = st.pending[rsp.id]
			if ictx == nil {
				fmt.Println("error: stubReceiver got no invocontext for id",
							rsp.id)
			} else {
				ictx.rch <- &rsp // reply back to the invoke() function
				st.pending[rsp.id] = nil, false // forget the invocontext
			}
		} else if err != nil {
			fmt.Println("stub: Got error decoding gob stream!:", err)
		}
	}
	fmt.Println("stubReceiver closed")
	st.ch <- &invocontext{invocation{quit, 0, nil}, nil} // quit msg
}

// skel base type (server side)
type Skel struct {
	rwc   io.ReadWriteCloser 	// The conn. or pipe to read/write gobs
	quit  chan int				// The quit channel (just like in the stub)
	e     *gob.Encoder			// Gobs encoder
	d     *gob.Decoder			// gobs decoder
	alive bool					// 'Is the skel alive' flag
	rch   chan *response		// reply channel
	x     executor				// executor will execute the remotized interface
}

// skel base 'constructor'
func NewSkel(rwc io.ReadWriteCloser, x executor) *Skel {
	sk := &Skel{rwc, make(chan int),
		gob.NewEncoder(rwc), gob.NewDecoder(rwc), true, make(chan *response), x}
	go skelReplier(sk)	
	go skelReceiver(sk)
	return sk
}

// close the skel by closing first the reader at skelReceiver() and waiting 
// for the quit signal from the skelReplier() 
func (sk *Skel) close() {
	if sk.alive {
		sk.alive = false
		sk.rwc.Close()
		<-sk.quit
	}
}

// Construcs/prepares a response to an invocation
func newResponseTo(i *invocation) *response {
	return &response{i.id, nil, nil}
}

// skel invocation receiver gets invocation gobs and launches a goroutine to
// execute them. The reply will go back via channel's msg to the skelReplier()
func skelReceiver(sk *Skel) {
	id := 0
	for sk.alive {
		var i invocation
		var rsp *response
		err := sk.d.Decode(&i)
		if err == os.EOF { // EOF?
			if(sk.alive) { // got quit signal by colsing the reader
				fmt.Println("skel warn: skelReceiver remotely stopped!")
				sk.alive = false // remote quit
			}
			continue // quit
		}
		if err == nil {
			id = i.id
			go func(i *invocation) { // execute invocation
				rsp := newResponseTo(i)
				rsp.results = sk.x.execute(i.funcNum, i.args)
				fmt.Println("skel: execution renders ", rsp)
				sk.rch <- rsp
			}(&i)
		} else { // report error back
			id++
			rsp = &response{id, nil, err}
			sk.rch <- rsp
		}
	}
	fmt.Println("skelReceiver() closed")
	sk.rch <- &response{quit, nil, nil} // quit skelReplier
}

// skel replier send the executions results or error gobs back to the 
// client stub
func skelReplier(sk *Skel) {
	for sk.alive {
		rsp := <-sk.rch
		if rsp.id == quit { // got quit signal
			continue // quit
		}
		err := sk.e.Encode(rsp) // sent respone back to the client stub
		if err != nil { // sent error repling...
			sk.e.Encode(&response{rsp.id, nil, err})
		}
	}
	fmt.Println("skelReplier closed")
	sk.quit <- 1
}

