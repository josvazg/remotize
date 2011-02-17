package riisample

import (
	"rii"
	"os"
)

type calcserver struct {
	s *rii.Server
}

func (cs *calcserver) add(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(1,op1,op2)
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cs *calcserver) subtract(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(2,op1,op2)	
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cs *calcserver) multiply(op1 float, op2 float) (res float, e os.Error) {
	rp,e:=cs.s.Invoke(3,op1,op2)
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cs *calcserver) divide(op1 float, op2 float) (res float, e os.Error) {
	rp,e=cs.s.Invoke(4,op1,op2)	
	r:=*rp
	if(rp!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

