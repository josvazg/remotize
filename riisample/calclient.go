package riisample

import (
	"rii"
	"os"
	"io"
)

type calcclient struct {
	riic *rii.Client
}

func newCalcClient(rwc io.ReadWriteCloser) Calc {
	return (interface{})(&calcclient{rii.NewClient(rwc)}).(Calc)
}

func (c *calcclient) add(op1 float, op2 float) (res float, e os.Error) {
	r,e:=c.riic.Call(1,op1,op2)
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (c *calcclient) subtract(op1 float, op2 float) (res float, e os.Error) {
	r,e:=c.riic.Call(2,op1,op2)	
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (c *calcclient) multiply(op1 float, op2 float) (res float, e os.Error) {
	r,e:=c.riic.Call(3,op1,op2)
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (c *calcclient) divide(op1 float, op2 float) (res float, e os.Error) {
	r,e:=c.riic.Call(4,op1,op2)	
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}


