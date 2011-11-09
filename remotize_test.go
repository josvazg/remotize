package remotize

import (
	"reflect"
	"rpc"
	"testing"
)

type Sometyper interface {

}

type RemoteSometyper struct {

}

type SometyperService struct {

}

func checkType(t *testing.T, typename string, i interface{}) {
	if i == nil {
		t.Fatal("Could not retrieve " + typename + "!")
	}
	if atype := reflect.TypeOf(i); atype.Kind() != reflect.Ptr || atype.Elem().Name() != typename {
		t.Fatal("Expected type '" + typename + "' but got '" +
			atype.Elem().Name() + "'!")
	}
}

func TestRegistry(t *testing.T) {
	Register(RemoteSometyper{}, 
		func (*rpc.Client) interface{} {
			return &RemoteSometyper{}
		},
		SometyperService{}, 
		func (interface{}) interface{} {
			return &SometyperService{}
		})
	s := NewService(new(Sometyper))
	checkType(t, "SometyperService", s)
	r := NewRemote(nil, new(Sometyper))
	checkType(t, "RemoteSometyper", r)
}

