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

var stubs mapper.Mapper

type skeletor interface {
	commonSkel() *commonSkel
	execute(funcNum int, args []interface{}) []interface{}
}

type call struct {
	id      int
	funcNum int
	args    []interface{}
}

type invocation struct {
	call
	rch chan *response
}

type response struct {
	id      int
	results []interface{}
	error   os.Error
}

type commonStub struct {
	e      *gob.Encoder
	d      *gob.Decoder
	alive    bool
	iface    *reflect.InterfaceType
	url      string
	ch       chan *invocation
	id2invok map[int]*invocation
}

func newStubBase(url string, rw io.ReadWriter) *commonStub {
	return &commonStub{gob.NewEncoder(rw), gob.NewDecoder(rw), true,
		nil, url, make(chan *invocation), make(map[int]*invocation)}
}

type commonSkel struct {
	e   *gob.Encoder
	d   *gob.Decoder
	alive bool
	rch   chan *response
}

func newSkelBase(rw io.ReadWriter) *commonSkel {
	return &commonSkel{gob.NewEncoder(rw), gob.NewDecoder(rw), true,
		make(chan *response)}
}

func init() {
	stubs = mapper.NewMapper(true, true, nil)
}

func (s *commonStub) invoke(funcNum int, args ...interface{}) (results *[]interface{}, err os.Error) {
	rch := make(chan *response)
	fmt.Println("Invoking ", funcNum, args, "...")
	s.ch <- &invocation{call{0, funcNum, args}, rch}
	fmt.Println("waiting response...")
	rsp := <-rch
	fmt.Println("done rsp:", rsp)
	if rsp.error != nil {
		return nil, rsp.error
	}
	return &rsp.results, nil
}

func (s *commonStub) setInvocation(invok *invocation) {
	s.id2invok[invok.id] = invok
}

func (s *commonStub) invocationFor(id int) *invocation {
	return s.id2invok[id]
}

func (s *commonStub) startStub() {
	go responseLoop(s)
	go invocationsLoop(s)
}

func (s *commonStub) stopStub() {
	s.alive = false
}

func invocationsLoop(s *commonStub) {
	fmt.Println("stub: invocationLoop Started")
	id := 0
	for s.alive {
		invok := <-s.ch
		fmt.Println("stub: Got invocation:", invok)
		id++
		invok.id = id
		fmt.Println("stub: Update invocation id:", invok)
		s.setInvocation(invok)
		fmt.Println("stub: Encode call:", invok.call)
		err := s.e.Encode(invok.call)
		if err != nil {
			fmt.Println("Encode err:", err)
			invok.rch <- &response{id, nil, err}
		} else {
			fmt.Println("Done call#", invok.id, ":", invok.call)
		}
	}
	fmt.Println("stub: invocationLoop Ended")
}

func responseLoop(s *commonStub) {
	fmt.Println("stub: responseLoop Started")
	for s.alive {
		var rsp *response
		error := s.d.Decode(&rsp)
		invok := s.invocationFor(rsp.id)
		if error != nil {
			invok.rch <- &response{invok.id, nil, error}
		} else {
			invok.rch <- rsp
		}
	}
	fmt.Println("stub: responseLoop Ended")
}

func (s *commonSkel) commonSkel() *commonSkel {
	return s
}

func startSkel(sk skeletor) {
	go callsLoop(sk)
	go replyLoop(sk.commonSkel())
}

func stopSkel(sk skeletor) {
	sk.commonSkel().alive = false
}

func (s *commonSkel) newResponseTo(rcall *call) *response {
	return &response{rcall.id, nil, nil}
}

func callsLoop(sk skeletor) {
	fmt.Println("skel: callsLoop Started")
	id := 0
	s := sk.commonSkel()
	for s.alive {
		var rcall call
		var rsp *response
		err := s.d.Decode(&rcall)
		if err == nil {
			id = rcall.id
			fmt.Println("got call id ", id)
			go func(rcall *call) {
				rsp := s.newResponseTo(rcall)
				rsp.results = s.execute(rcall.funcNum, rcall.args)
				s.rch <- rsp
			}(&rcall)
		} else {
			id++
			rsp = &response{id, nil, err}
			s.rch <- rsp
		}
	}
	fmt.Println("skel: callsLoop Ended")
}

func replyLoop(s *commonSkel) {
	fmt.Println("skel: replyLoop Started")
	for s.alive {
		rsp := <-s.rch
		err := s.e.Encode(rsp)
		if err != nil {
			s.e.Encode(&response{rsp.id, nil, err})
		}
	}
	fmt.Println("skel: replyLoop Ended")
}
