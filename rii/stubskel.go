package rii

import (
	"container/mapper"
	"reflect"
	"os"
	"gob"
)

var stubs mapper.Mapper

type Stubber interface {
	getInterfaceType() *reflect.InterfaceType
}

type commonStub struct {
	iface *reflect.InterfaceType
	url string
	enc *gob.Encoder
	dec *gob.Decoder
}

type commonSkel struct {
	iface *reflect.InterfaceType
	alive bool
	enc *gob.Encoder
	dec *gob.Decoder
}

type riiMsg struct {
	funcNum int
	data []interface{}
	error os.Error
}

func init() {
	stubs=mapper.NewMapper(true,true,new(Stubber))
}

func (s *commonStub) invoke(funcNum int, args ... interface{}) (results *[]interface{}, err os.Error) {
	var response riiMsg
	invocation:=riiMsg{funcNum,args,nil}
	s.enc.Encode(invocation)
	error:=s.dec.Decode(response)
	if(error!=nil) {
		return nil,error
	}
	if(response.error!=nil) {
		return nil,response.error
	}
	return &response.data,nil
}

func (s *commonSkel) callsLoop() {
	for ;s.alive; {
		var invocation,response *riiMsg
		err:=s.dec.Decode(invocation);
		if(err!=nil) {
			response=&riiMsg{-1,nil,err}			
		} else {
			response=s.execute(invocation)
		}
		s.enc.Encode(response)		
	}
}

func (s *commonSkel) execute(invocation *riiMsg) *riiMsg {
	// nothing in the common implementation
	return nil
}












