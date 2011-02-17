package riisample

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

const (
	RIIGOBSTART="rii:gob start\n"
)


type Calc interface {
	add(op1 float, op2 float) (float, os.Error)
	subtract(op1 float, op2 float) (float, os.Error)
	multiply(op1 float, op2 float) (float, os.Error)
	divide(op1 float, op2 float) (float, os.Error)
}

func gobSync(r io.Reader,w io.Writer) {
	fmt.Fprintf(w,RIIGOBSTART)
	markSize:=len(RIIGOBSTART)
	buf:=make([]byte,markSize)
	found:=false
	n,e:=r.Read(buf)
	matched:=0
	for s:=0;n>0 && e==nil && !found; {
		for i:=s;i<markSize && matched<markSize;i++ {
			if(buf[i]==RIIGOBSTART[matched]) {
				matched++
			} else if(matched>0) {
				matched=0
				i--
			}
		}
		if(matched==markSize) {
			found=true
			break
		} else if(matched>0) {
			for i:=0;i<matched;i++ {
				buf[i]=buf[markSize-matched+i]
			}
			s=matched
		} else {
			s=0
		}
		n,e=r.Read(buf[s:])
	}
}

type rdebug struct {
	label	string
	r		io.ReadCloser	
	log		io.Writer
}

func DebugReader(label string,r io.ReadCloser, log io.Writer) *rdebug {
	return &rdebug{label,r,log}
}

func (rd *rdebug) Read(b []byte) (int,os.Error) {
	n,e:=rd.r.Read(b)
	if(e!=nil) {
		fmt.Fprintf(rd.log,"%v error receiving: %v\n",rd.label,e)	
	} else if(n>0) {
		fmt.Fprintf(rd.log,"%v readed %vbytes: %v '%v'\n",rd.label,n,b[:n],string(b[:n]))
	} else {
		fmt.Fprintf(rd.log,"%v readed: <nil msg>\n",rd.label)
	}
	return n,e
}

func (rd *rdebug) Close() os.Error {
	return rd.r.Close()
}

type wdebug struct {
	label	string
	w		io.Writer	
	log		io.Writer
}

func DebugWriter(label string,w io.Writer, log io.Writer) *wdebug {
	return &wdebug{label,w,log}
}

func (wd *wdebug) Write(b []byte) (int,os.Error) {
	n,e:=wd.w.Write(b)
	if(e!=nil) {
		fmt.Fprintf(wd.log,"%v error writing: %v\n",wd.label,e)	
	} else if(n>0) {
		fmt.Fprintf(wd.log,"%v written %vbytes: %v '%v'\n",wd.label,n,b[:n],string(b[:n]))
	} else {
		fmt.Fprintf(wd.log,"%v written: <nil msg>\n",wd.label)
	}
	return n,e
}

type simplecalc struct {
	r float
}

func (sc *simplecalc) add(op1 float, op2 float) (float, os.Error) {
	sc.r=op1+op2
	return sc.r,nil
}

func (sc *simplecalc) subtract(op1 float, op2 float) (float, os.Error) {
	sc.r=op1-op2
	return sc.r,nil
}

func (sc *simplecalc) multiply(op1 float, op2 float) (float, os.Error) {
	sc.r=op1*op2
	return sc.r,nil
}

func (sc *simplecalc) divide(op1 float, op2 float) (float, os.Error) {
	if(op2==0) {
		return 0,os.NewError("Divide "+strconv.Ftoa(op1,'f',-1)+" by ZERO!?!")
	}
	sc.r=op1/op2
	return sc.r,nil
}


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

