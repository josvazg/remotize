/*

	stubskel_test.go 	simulates a call though a stub and a skel

*/
package rii

import (
	"testing"
	"fmt"
	"io"
	"os"
	"time"
)

type memsocket struct {
	r io.ReadCloser
	w io.WriteCloser
}

func newMemSocket(r io.ReadCloser, w io.WriteCloser) (*memsocket) {
	return &memsocket{r,w}
}

func (ms *memsocket) Read(p []byte) (n int, err os.Error) {
	return ms.r.Read(p)
}

func (ms *memsocket) Write(p []byte) (n int, err os.Error) {
	return ms.w.Write(p)
}

func (ms *memsocket) Close() os.Error {
	ms.w.Close()
	ms.r.Close()
	return nil
}

type someskeletor int;

func (s *someskeletor) execute(funcNum int,args []interface{}) []interface{} {
	switch {
	case funcNum==1:
		arg1:=(args[0]).(int)
		arg2:=(args[1]).(int)
		var results []interface{}=make([]interface{},1)
		results[0]=s.add(arg1,arg2)
		return results
	case funcNum==2:
		arg1:=(args[0]).(int)
		arg2:=(args[1]).(int)
		var results []interface{}=make([]interface{},1)
		results[0]=s.addnsleep(arg1,arg2)
		return results		
	}		
	return nil
}

func (s *someskeletor) add(a int, b int) int {
	*s=someskeletor(a+b);
	return int(*s);
}

func (s *someskeletor) addnsleep(a int, b int) int {
	*s=someskeletor(a+b);
	r:=int(*s)
	sleep:=int64(r)*1e7
	fmt.Printf("Sleep=%vms\n",(sleep/1e6))
	time.Sleep(sleep)
	return r;
}

func TestStubSkel(t *testing.T) {
	r1,w1:=io.Pipe()
	r2,w2:=io.Pipe()
	localSocket:=newMemSocket(r1, w2)
	remoteSocket:=newMemSocket(r2, w1)
	st:=newStub("local:",localSocket)
	var somes someskeletor
	sk:=newSkel(remoteSocket,&somes)
	fn:=1
	arg1:=2
	arg2:=7
	fmt.Println("invoking function#",fn," with args:",arg1,arg2)
	res,err:=st.invoke(fn,arg1,arg2)
	fmt.Println("TEST RESULT:(",arg1,")+(",arg2,")=",(*res)[0]," err:",err)
	st.close()
	sk.close()
}

func Test3Calls(t *testing.T) {
	quit:=make(chan int)
	r1,w1:=io.Pipe()
	r2,w2:=io.Pipe()
	localSocket:=newMemSocket(r1, w2)
	remoteSocket:=newMemSocket(r2, w1)
	st:=newStub("local:",localSocket)
	var somes someskeletor
	sk:=newSkel(remoteSocket,&somes)
	go func() {
		fn:=2
		arg1:=2
		arg2:=7
		fmt.Println("invoking function#",fn," with args:",arg1,arg2)
		res,err:=st.invoke(fn,arg1,arg2)
		fmt.Println("TEST 1 RESULT:(",arg1,")+(",arg2,")=",(*res)[0]," err:",err)
		quit<-1
	}()
	go func() {
		fn:=2
		arg1:=1
		arg2:=3
		fmt.Println("invoking function#",fn," with args:",arg1,arg2)
		res,err:=st.invoke(fn,arg1,arg2)
		fmt.Println("TEST 2 RESULT:(",arg1,")+(",arg2,")=",(*res)[0]," err:",err)
		quit<-1
	}()
	go func() {
		fn:=2
		arg1:=1
		arg2:=1
		fmt.Println("invoking function#",fn," with args:",arg1,arg2)
		res,err:=st.invoke(fn,arg1,arg2)
		fmt.Println("TEST 3 RESULT:(",arg1,")+(",arg2,")=",(*res)[0]," err:",err)
		quit<-1
	}()
	<-quit
	<-quit
	<-quit
	st.close()
	sk.close()
}

