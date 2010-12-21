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
	var results []interface{}=make([]interface{},1)
	results[0]=0
	return results
}

type rwbuffer struct {
	buf []byte
	r,w int 
	written chan int
	reading chan int
}

func (rw *rwbuffer) Read(p []byte) (n int, err os.Error) {
	fmt.Println("rw.r=",rw.r," rw.w=",rw.w)	
	fmt.Println("reading into p=",p)	
	if(rw.r==rw.w) {
		fmt.Println("Waiting write...")
		<-rw.written
		fmt.Println("Writen...")
	}
	ln:=rw.w-rw.r
	fmt.Println("ln=",ln)
	if(ln==0) {
		fmt.Println("NO READ")
		return 0,nil
	}
	if(ln>len(p)) {
		ln=len(p)
		fmt.Println("*ln=",ln)
	}
	s:=rw.r
	rw.r+=ln
	for i:=0;i<ln;i++ {
		p[i]=rw.buf[s+i]
	}
	fmt.Println("read p=",p)
	return ln,nil
}

func (rw *rwbuffer) Write(p []byte) (n int, err os.Error) {
	fmt.Println("writting p=",p)
	if(rw.r==rw.w) {
		rw.buf=make([]byte,len(p))
		copy(rw.buf,p)
		rw.r=0
		rw.w=len(rw.buf)
	} else {
		rw.buf=append(rw.buf,p...)
		rw.w+=len(rw.buf)
        }
	fmt.Println("in buf=",rw.buf)
	_,reading:=<-rw.reading
	if(reading) {
		rw.written<-1
	}
	return len(p),nil
}

func newRW() (*rwbuffer) {
	return &rwbuffer{nil,0,0,make(chan int),make(chan int)}
}

type sometype struct {
	n int
	name string
}

func TestStubSkel(t *testing.T) {
	/*r1,w1:=io.Pipe()
	r2,w2:=io.Pipe()
	localSocket:=newMemSocket(r1, w2)
	remoteSocket:=newMemSocket(r2, w1)*/
	quit:=make(chan int)
	rw:=newRW()
	go func() {
		/*bytes:=make([]byte,10)	
		n,_:=remoteSocket.Read(bytes)
		fmt.Println("L->R Recv",n,"bytes:",string(bytes))*/
		d:=gob.NewDecoder(rw)
		str:=sometype{0,""}
		d.Decode(str)
		/*buf:=make([]byte,100)
		n,_:=rw.Read(buf)
		fmt.Println("Recv:",buf[:n])*/
		<-quit
	}()
	//localSocket.Write([]byte("Hola"))
	e:=gob.NewEncoder(rw)
	stw:=sometype{1,"hola"}
	e.Encode(stw)
	<-quit
	//fmt.Println("rw.buf",rw.buf)
	/*sb:=newStubBase("local:",localSocket)
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
	stopSkel(sk)*/
}


