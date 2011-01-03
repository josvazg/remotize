package riisample

import (
	"testing"
	"fmt"
	"os"
	"exec"
	"rii"
	"time"
)

type stringreader struct {
	s string
	pos int
}

func (r *stringreader) Read(b []byte) (int,os.Error) {
	slen:=len(r.s)
	if(r.pos>=slen) {
		return 0,os.EOF
	}
	n:=slen-r.pos
	blen:=len(b)
	if(n>blen) {
		n=blen
	}
	for i:=0;i<n;i++ {
		b[i]=r.s[r.pos+i]
	}
	r.pos+=n
	return n,nil
}

func TestGobSync(t *testing.T) {
	//  0123456 789012345678901234567890123456789 0123456
	s:="ksdjfh\njkwefgrii:goarerwerrii:gob start\ndfjkrgh"
	fmt.Println("len(s)=",len(s))
	sr:=&stringreader{s,0}
	gobSync(sr,os.Stdout)
	pos:=sr.pos-len(RIIGOBSTART)
	fmt.Print("found at ",pos," '",sr.s[pos:sr.pos-1],"'\n")	
}

func TestRiisample(t *testing.T) {
	fmt.Println("Hi!")
	os.Setenv("RUN_RIISAMPLE_SERVER","RUN")
	argv := []string{"gotest", "server_test.go"} 
	cmdname, e := exec.LookPath(argv[0]) 
	if e != nil { 
		t.Fatal("exec %s: %s", argv[0], e) 
  	}
	cmd,err:=exec.Run(cmdname,argv,os.Environ(),"",exec.Pipe,exec.Pipe,
		exec.PassThrough)
	if err !=nil {
		t.Fatal("3 %v",err) 
  	}
	fmt.Println("Cmd pid",cmd.Pid)
	gobSync(cmd.Stdout,cmd.Stdin)
	/*cs:=&calcstub{rii.NewStub(DebugReader("cmd.Stdout",cmd.Stdout,os.Stderr),
		DebugWriter("cmd.Stdin",cmd.Stdin,os.Stderr))}*/
	cs:=&calcstub{rii.NewStub(cmd.Stdout,cmd.Stdin)}
	calc:=(interface{})(cs).(Calc)
	r,_:=calc.add(1,2.3)
	fmt.Println("1+2.3=",r)
	r,_=calc.subtract(1,2.3)
	fmt.Println("1-2.3=",r)
	r,_=calc.multiply(4.5,2.1)
	fmt.Println("4.5*2.1=",r)
	r,_=calc.divide(3,5.2)
	fmt.Println("3/52.3=",r)
	lcalc:=&simplecalc{0}
	t1:=time.Nanoseconds()
	for i:=0;i<1000;i++ {
		lcalc.add(1,2.3)
		lcalc.subtract(1,2.3)
		lcalc.multiply(4.5,2.1)
		lcalc.divide(3,5.2)
	}
	lt:=float(time.Nanoseconds()-t1)/1e6
	fmt.Print("1000 LOCAL calc iterations in ",lt,"ms\n")
	t2:=time.Nanoseconds()
	for i:=0;i<1000;i++ {
		calc.add(1,2.3)
		calc.subtract(1,2.3)
		calc.multiply(4.5,2.1)
		calc.divide(3,5.2)
	}
	rt:=float(time.Nanoseconds()-t2)/1e6
	fmt.Print("1000 remote calc iterations in ",rt,"ms\n")
	fmt.Print("Remote is ",(rt/lt),"times slower than local\n")
	os.Setenv("RUN_RIISAMPLE_SERVER","")
}
