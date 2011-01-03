package riisample

import (
	"rii"
	"os"
)

type calcstub struct {
	s *rii.Stub
}

func (cs *calcstub) add(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(1,op1,op2)
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cs *calcstub) subtract(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(2,op1,op2)	
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cs *calcstub) multiply(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(3,op1,op2)	
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cs *calcstub) divide(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(4,op1,op2)	
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}


