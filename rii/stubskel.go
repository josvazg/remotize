// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// rii package is the Remote Interface Invocation foundation allowing go 
// programs to use out-of-process services defined by an interface, either
// locally or remotelly without worring (too much or too soon) 
// about the communications. With this package you can remotize local parts of 
// the program or load them dynamically as a plugin.
package rii

import (
	"container/mapper"
	"reflect"
	"os"
	"io"
	"gob"
	"fmt"
)

const (
	quit=-1
)

var stubs mapper.Mapper

type skeletor interface {
	execute(funcNum int, args []interface{}) []interface{}
}

type invocation struct {
	id      int
	funcNum int
	args    []interface{}
}

type invocontext struct {
	invocation
	rch chan *response
}

type response struct {
	id      int
	results []interface{}
	error   os.Error
}

type stub struct {
	rwc	io.ReadWriteCloser
	quit	chan int
	e	*gob.Encoder
	d      *gob.Decoder
	alive    bool
	iface    *reflect.InterfaceType
	url      string
	ch       chan *invocontext
	id2ictx map[int]*invocontext
}

func newStub(url string, rwc io.ReadWriteCloser) *stub {
	st:=&stub{rwc,make(chan int),
		gob.NewEncoder(rwc), gob.NewDecoder(rwc), true,
		nil, url, make(chan *invocontext), make(map[int]*invocontext)}
	go returnIn(st)
	go invocOut(st)
	return st
}

func init() {
	stubs = mapper.NewMapper(true, true, nil)
}

func (st *stub) invoke(funcNum int, args ...interface{}) (results *[]interface{}, err os.Error) {
	rch := make(chan *response)
	fmt.Println("Invoking ", funcNum, args, "...")
	st.ch <- &invocontext{invocation{0, funcNum, args}, rch}
	fmt.Println("waiting response...")
	rsp := <-rch
	fmt.Println("done rsp:", rsp)
	if rsp.error != nil {
		return nil, rsp.error
	}
	return &rsp.results, nil
}

func (st *stub) close() {
	if(st.alive) {
		fmt.Println("close stub")	
		st.alive = false
		st.rwc.Close()
		<-st.quit
	}
}

func invocOut(st *stub) {
	fmt.Println("stub: invocOut Started")
	id := 0
	for st.alive {
		ictx := <-st.ch
		if ictx.id==quit {
			continue; // quit
		}
		fmt.Println("stub: Got invocontext:", ictx)
		id++
		ictx.id = id
		fmt.Println("stub: Update invocontext id:", ictx)
		st.id2ictx[ictx.id]=ictx // remember invocation context
		fmt.Println("stub: (+)pending:",st.id2ictx)
		fmt.Println("stub: Encode invocation:", ictx.invocation)
		err := st.e.Encode(ictx.invocation)
		if err != nil {
			fmt.Println("stub: Encode err:", err)
			ictx.rch <- &response{id, nil, err}
		} else {
			fmt.Println("stub: Done call#", ictx.id, ":", ictx.invocation)
		}
	}
	fmt.Println("stub: invocOut Ended")
	st.quit<-1
}

func returnIn(st *stub) {
	fmt.Println("stub: returnIn Started")
	for st.alive {
		var rsp response
		var ictx *invocontext
		err := st.d.Decode(&rsp)
		if(!st.alive) {
			fmt.Println("stub: returnIn Stopped!")
			continue; // quit
		}
		if err==os.EOF {
			fmt.Println("stub: returnIn remotely Stopped!")
			st.alive=false
			continue; // quit
		}
		if err==nil {
			ictx = st.id2ictx[rsp.id]
			if(ictx==nil) {
				fmt.Println("stub: No invocation context for id",rsp.id)
			} else {
				ictx.rch<-&rsp
				st.id2ictx[rsp.id]=nil,false
				fmt.Println("stub: (-)pending:",st.id2ictx)
			}
		} else if(err!=nil) {
			fmt.Println("stub: Got error decoding gob stream!:",err)
		}		
	}
	fmt.Println("stub: returnIn Ended")
	st.ch<-&invocontext{invocation{quit, 0, nil}, nil} // quit msg
}

type skel struct {
	rwc	io.ReadWriteCloser
	quit	chan int
	e   *gob.Encoder
	d   *gob.Decoder
	alive bool
	rch	chan *response
	s skeletor 
}

func newSkel(rwc io.ReadWriteCloser, s skeletor) *skel {
	sk:=&skel{rwc,make(chan int),
		gob.NewEncoder(rwc), gob.NewDecoder(rwc), true,make(chan *response),s}
	go invocIn(sk)
	go returnOut(sk)
	return sk
}

func (sk *skel) close() {
	if(sk.alive) {
		fmt.Println("close skel")
		sk.alive = false
		sk.rwc.Close()
		<-sk.quit
	}
}

func newResponseTo(i *invocation) *response {
	return &response{i.id, nil, nil}
}

func invocIn(sk *skel) {
	fmt.Println("skel: invocIn Started")
	id := 0
	for sk.alive {
		var i invocation
		var rsp *response
		err := sk.d.Decode(&i)
		if(!sk.alive) {
			fmt.Println("skel: invocIn Stopped!")
			continue; // quit
		}
		if err==os.EOF {
			fmt.Println("skel: invocIn remotely Stopped!")
			sk.alive=false
			continue; // quit
		}
		if err == nil {
			id = i.id
			fmt.Println("skel: incoming invocation id ", id)
			go func(i *invocation) {
				rsp := newResponseTo(i)
				rsp.results = sk.s.execute(i.funcNum, i.args)
  			    fmt.Println("skel: execution renders ", rsp)
				sk.rch <- rsp
			}(&i)
		} else {
			id++
			rsp = &response{id, nil, err}
			sk.rch <- rsp
		}
	}
	fmt.Println("skel: invocIn Ended")
	sk.rch<-&response{quit,nil,nil}
}

func returnOut(sk *skel) {
	fmt.Println("skel: returnOut Started")
	for sk.alive {
		rsp := <-sk.rch
		if rsp.id==quit {
			fmt.Println("skel: returnOut Stopped!")
			continue; // quit
		}
		err := sk.e.Encode(rsp)
		if err != nil {
			sk.e.Encode(&response{rsp.id, nil, err})
		} else {
  			fmt.Println("skel: reply sent back", rsp)
		}
	}
	fmt.Println("skel: returnOut Ended")
	sk.quit<-1
}

