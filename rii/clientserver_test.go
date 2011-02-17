package rii

import (
	"io"
	"os"
	"strconv"
	"testing"
)

// The interface
type Calc interface {
	Add(op1 float64, op2 float64) float64
	Subtract(op1 float64, op2 float64) float64
	Multiply(op1 float64, op2 float64) float64
	Divide(op1 float64, op2 float64) (float64, os.Error)
}

// The implementation
type simplecalc struct {
	r float64
}

func (sc *simplecalc) Add(op1 float64, op2 float64) float64 {
	sc.r=op1+op2
	return sc.r
}

func (sc *simplecalc) Subtract(op1 float64, op2 float64) float64 {
	sc.r=op1-op2
	return sc.r
}

func (sc *simplecalc) Multiply(op1 float64, op2 float64) float64 {
	sc.r=op1*op2
	return sc.r
}

func (sc *simplecalc) Divide(op1 float64, op2 float64) (float64, os.Error) {
	if(op2==0) {
		return 0,os.NewError("Divide "+strconv.Ftoa64(op1,'f',-1)+" by ZERO!?!")
	}
	sc.r=op1/op2
	return sc.r,nil
}

type CalcServer struct {
	s *Server
}

// The Server wiring
func (s *CalcServer) Add(a []interface{},r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	i:=s.s.Iface.(Calc)
	res[0]=i.Add((a[0]).(float64),(a[1]).(float64))
	res[1]=nil
	r=&res
	return nil
}

func (s *CalcServer) Subtract(a []interface{}, r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	i:=s.s.Iface.(Calc)
	res[0]=i.Subtract((a[0]).(float64),(a[1]).(float64))
	res[1]=nil
	r=&res
	return nil
}

func (s *CalcServer) Multiply(a []interface{}, r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	i:=s.s.Iface.(Calc)
	res[0]=i.Multiply((a[0]).(float64),(a[1]).(float64))
	res[1]=nil
	r=&res
	return nil
}

func (s *CalcServer) Divide(a []interface{}, r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	i:=s.s.Iface.(Calc)
	res[0],res[1]=i.Divide((a[0]).(float64),(a[1]).(float64))
	r=&res
	return (res[1]).(os.Error)
}

// The Client wiring
type calcClient struct {
	c	*Client
}

func (cc *calcClient) add(op1 float64, op2 float64) float64 {
	r,e:=cc.c.Call("Calc.Add",false,op1,op2)
	if(e!=nil) {
		cc.c.HandleError("add",e)
	}	
	return
}

func (cc *calcClient) subtract(op1 float64, op2 float64) float64 {
	r,e:=c.riic.Call("Calc.subtract",false,op1,op2)	
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float)
	}	
	return
}

func (cc *calcClient) multiply(op1 float64, op2 float64) float64 {
	r,e:=c.riic.Call("Calc.Multiply",false,op1,op2)
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float64)
	}	
	return
}

func (cc *calcClient) divide(op1 float64, op2 float64) (float64, os.Error) {
	r,e:=c.riic.Call("Calc.Divide",true,op1,op2)	
	if(r!=nil && len(r)>0) {
		res=(r[0]).(float64)
	}	
	return
}

func TestClientServerLocal(t *testing.T) {
	s:=NewServer(&simplecalc{0})
	r1,w1:=io.Pipe()
	r2,w2:=io.Pipe()
	s.Add(doAdd)
	s.Add(doSubtract)
	s.Add(doMultiply)
	s.Add(doDivide)
	s.ServePipe(IO(r2,w1))
	c:=NewClient(IO(r1,w2))
	cc:=&calcClient{c}
	
}
