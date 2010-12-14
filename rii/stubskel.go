/*

	stubskel.go 	implements the common stub and skel core functions for 
	Remote Interface Invocations

*/
package rii

import (
	"container/mapper"
	"reflect"
	"os"
	"io"
	"gob"
)

var stubs mapper.Mapper

type skeletor interface {
	execute(*invocation) *response
}

type response struct {
	id int
	results []interface{}
	error os.Error
}

type invocation struct {
	id int
	funcNum int
	args []interface{}
	rch chan *response
}

type commonStubSkel struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

type commonStub struct {
	commonStubSkel
	iface *reflect.InterfaceType
	url string
	alive bool
	ch chan *invocation
}

type commonSkel struct {
	commonStubSkel
	iface *reflect.InterfaceType
	alive bool
}

type StubSkeletor interface {
	alive() bool
	enc() *gob.Encoder
	dec() *gob.Decoder
	invocationFor(int) *invocation
}

type Stubber interface {
	StubSkeletor
	ch() chan *invocation
}

type Skeletor interface {
	StubSkeletor
	execute(*invocation) *response
	rch() chan *response
}

func init() {
	stubs=mapper.NewMapper(true,true,new(Stubber))
}

func newStubBase(url string, iface *reflect.InterfaceType, 
		rw io.ReadWriter) (*commonStub) {
	return &commonStub{commonStubSkel{gob.NewEncoder(rw),
		gob.NewDecoder(rw)},iface,url,false,make(chan *invocation)}
}

func (s *commonStub) invoke(funcNum int, 
		args ... interface{}) (results *[]interface{}, err os.Error) {
	rch:=make(chan *response)
	s.ch<-&invocation{0,funcNum,args,rch}
	rsp:=<-rch
	if(rsp.error!=nil) {
		return nil,rsp.error
	}
	return &rsp.results,nil
}

func invocationsLoop(s Stubber) {
	id:=0
	for ;s.alive(); {
		invok:=<-s.ch()
		id++
		invok.id=id
		err:=s.enc().Encode(invok)
		if(err!=nil) {
			// TODO local error 
		}
	}
}

func responseLoop(s Stubber) {
	for ;s.alive(); {
		var rsp *response
		error:=s.dec().Decode(rsp)
		invok:=s.invocationFor(rsp.id)
		if(error!=nil) {
			invok.rch<-&response{invok.id,nil,error}
		} else {
			invok.rch<-rsp
		}
	}
}

func callsLoop(s Skeletor) {
	id:=0
	for ;s.alive(); {
		var invok *invocation
		err:=s.dec().Decode(invok)
		id=invok.id
		var rsp *response
		if(err!=nil) {
			rsp=&response{id,nil,err}
			s.rch()<-rsp
		} else {
			go func(*invocation) {
				rsp:=s.execute(invok)
				s.rch()<-rsp
			}(invok)
		}
	}
}

func replyLoop(s Skeletor) {
	for ;s.alive(); {
		rsp:=<-s.rch()
		err:=s.enc().Encode(rsp)
		if(err!=nil) {
			s.enc().Encode(&response{rsp.id,nil,err})
		}
	}
}


