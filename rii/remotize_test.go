package rii

import (
	"testing"
	"fmt"
	"io"
	"reflect"
)

func TestRemotize(t *testing.T) {
	var i *io.ReadWriteCloser
	fmt.Println("(void pointer)...")
	Remotize(i)
	fmt.Println("new(io.ReadWriteCloser)")
	Remotize(new(io.ReadWriteCloser))
	fmt.Println("reflect.Typeof(new(io.ReadWriteCloser)).(*reflect.PtrType).Elem()...")
	Remotize(reflect.Typeof(new(io.ReadWriteCloser)).(*reflect.PtrType).Elem())
	fmt.Println("new(interface{})...")
	Remotize(new(interface{}))
}
