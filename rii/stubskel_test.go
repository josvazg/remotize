package rii

import (
	"testing"
	"fmt"
	"io"
	"time"
)

type adder interface {
	add(int, int) int
	addnsleep(int, int) int
}

type addertype struct {
	i int
}

func (at* addertype) add(a int, b int) int {
	at.i = a + b
	return at.i
}

func (at* addertype) addnsleep(a int, b int) int {
	at.i = a + b
	r := at.i
	sleep := int64(r) * 1e7
	fmt.Printf("Sleep=%vms\n", (sleep / 1e6))
	time.Sleep(sleep)
	return r
}

func f1add(iface interface{}, args*[]interface{}) *[]interface{} {
	fmt.Println("iface=",iface)
	fmt.Println("args=",args)
	arg1 := ((*args)[0]).(int)
	arg2 := ((*args)[1]).(int)
	var results []interface{} = make([]interface{}, 1)
	results[0] = (iface).(adder).add(arg1, arg2)
	return &results
}

func f2addnsleep(iface interface{}, args*[]interface{}) *[]interface{} {
	arg1 := ((*args)[0]).(int)
	arg2 := ((*args)[1]).(int)
	var results []interface{} = make([]interface{}, 1)
	results[0] = (iface).(adder).addnsleep(arg1, arg2)
	return &results
}

func TestStubSkel(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	st := NewStub(r1,w2)
	var a addertype
	sk := NewSkel(r2, w1, &a)
	sk.Add(f1add)
	sk.Add(f2addnsleep)
	go sk.Serve()
	fn := 1
	arg1 := 2
	arg2 := 7
	fmt.Println("invoking function#", fn, " with args:", arg1, arg2)
	res, err := st.Invoke(fn, arg1, arg2)
	fmt.Println("TEST RESULT:(", arg1, ")+(", arg2, ")=", (*res)[0], " err:", err)
	st.close()
	sk.close()
}

func Test3Calls(t *testing.T) {
	quit := make(chan int)
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	st := NewStub(r1,w2)
	var a addertype
	sk := NewSkel(r2,w1, &a)
	sk.Add(f1add)
	sk.Add(f2addnsleep)
	go sk.Serve()
	go func() {
		fn := 2
		arg1 := 2
		arg2 := 7
		fmt.Println("invoking function#", fn, " with args:", arg1, arg2)
		res, err := st.Invoke(fn, arg1, arg2)
		fmt.Println("TEST 1 RESULT:(", arg1, ")+(", arg2, ")=", (*res)[0], " err:", err)
		quit <- 1
	}()
	go func() {
		fn := 2
		arg1 := 1
		arg2 := 3
		fmt.Println("invoking function#", fn, " with args:", arg1, arg2)
		res, err := st.Invoke(fn, arg1, arg2)
		fmt.Println("TEST 2 RESULT:(", arg1, ")+(", arg2, ")=", (*res)[0], " err:", err)
		quit <- 1
	}()
	go func() {
		fn := 2
		arg1 := 1
		arg2 := 1
		fmt.Println("invoking function#", fn, " with args:", arg1, arg2)
		res, err := st.Invoke(fn, arg1, arg2)
		fmt.Println("TEST 3 RESULT:(", arg1, ")+(", arg2, ")=", (*res)[0], " err:", err)
		quit <- 1
	}()
	<-quit
	<-quit
	<-quit
	st.close()
	sk.close()
}

