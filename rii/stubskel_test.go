/*

	stubskel_test.go 	simulates a call though a stub and a skel

*/
package rii

import (
	"testing"
	"fmt"
	"io"
	"os"
	"gob"
)

type memsocket struct {
	r io.Reader
	w io.Writer
}

func newMemSocket(r io.Reader, w io.Writer) (*memsocket) {
	return &memsocket{r,w}
}

func (ms *memsocket) Read(p []byte) (n int, err os.Error) {
	return ms.r.Read(p)
}

func (ms *memsocket) Write(p []byte) (n int, err os.Error) {
	return ms.w.Write(p)
}

func (s *commonSkel) execute(funcNum int,args []interface{}) []interface{} {
	if(funcNum==1) {
		arg1:=(args[0]).(int)
		arg2:=(args[1]).(int)
		var results []interface{}=make([]interface{},1)
		results[0]=add(arg1,arg2)
		return results
	}		
	return nil
}

func add(a int, b int) int {
	return a+b;
}

type sometype struct {
	n int
	name string
}

func TestStubSkel(t *testing.T) {
	r1,w1:=io.Pipe()
	r2,w2:=io.Pipe()
	localSocket:=newMemSocket(r1, w2)
	remoteSocket:=newMemSocket(r2, w1)
	go func() {
		stw:=sometype{1,"hola"}
		e:=gob.NewEncoder(localSocket)
		e.Encode(stw)
	}()
	var str sometype
	d:=gob.NewDecoder(remoteSocket)
	d.Decode(&str)
	fmt.Println("Recv:",str)	

	sb:=newStubBase("local:",localSocket)
	sk:=newSkelBase(remoteSocket)
	sb.startStub()
	startSkel(sk)
	fn:=1
	arg1:=2
	arg2:=-3
	fmt.Println("invoking function#",fn," with args:",arg1,arg2)
	res,ok:=sb.invoke(fn,arg1,arg2)
	fmt.Println("res:",res," ok:",ok)
	sb.stopStub()
	stopSkel(sk)
}


