/*

	stubskel_test.go 	simulates a call though a stub and a skel

*/
package rii

import (
	"testing"
	"fmt"
	"io"
	"os"
)

type memsocket struct {
	in *io.PipeReader
	out *io.PipeWriter
}

func newMemSocket(in *io.PipeReader, out *io.PipeWriter) (*memsocket) {
	return &memsocket{in,out}
}

func (ms *memsocket) Read(p []byte) (n int, err os.Error) {
	return ms.in.Read(p)
}

func (ms *memsocket) Write(p []byte) (n int, err os.Error) {
	return ms.out.Write(p)
}

func (s *commonSkel) execute(funcNum int,args []interface{}) []interface{} {
	var results []interface{}=make([]interface{},1)
	results[0]=0
	return results
}

func TestStubSkel(t *testing.T) {
	outBoundIn,outBoundOut:=io.Pipe()
	inBoundIn,inBoundOut:=io.Pipe()
	localSocket:=newMemSocket(inBoundIn, outBoundOut)
	remoteSocket:=newMemSocket(outBoundIn, inBoundOut)
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


