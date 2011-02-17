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

// The type implementing it
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

// The server reference-wrapping type
type CalcServer struct {
	i simplecalc
}

// The Server wiring
func (s *CalcServer) Add(a *[]interface{},r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	res[0]=s.i.Add(((*a)[0]).(float64),((*a)[1]).(float64))
	res[1]=nil
	r=&res
	return nil
}

func (s *CalcServer) Subtract(a *[]interface{}, r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	res[0]=s.i.Subtract(((*a)[0]).(float64),((*a)[1]).(float64))
	res[1]=nil
	r=&res
	return nil
}

func (s *CalcServer) Multiply(a *[]interface{}, r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	res[0]=s.i.Multiply(((*a)[0]).(float64),((*a)[1]).(float64))
	res[1]=nil
	r=&res
	return nil
}

func (s *CalcServer) Divide(a *[]interface{}, r *[]interface{}) os.Error {
	res:=make([]interface{},2)
	res[0],res[1]=s.i.Divide(((*a)[0]).(float64),((*a)[1]).(float64))
	r=&res
	return (res[1]).(os.Error)
}

// The Client reference wiring
type calcClient struct {
	c	*Client
}

func (cc *calcClient) Add(op1 float64, op2 float64) float64 {
	r,e:=cc.c.Call("Calc.Add",op1,op2)
	if(e!=nil) {
		cc.c.HandleError("Calc.Add",e)
	}
	return ((*r)[0]).(float64)
}

func (cc *calcClient) Subtract(op1 float64, op2 float64) float64 {
	r,e:=cc.c.Call("Calc.Subtract",op1,op2)	
	if(e!=nil) {
		cc.c.HandleError("Calc.Substract",e)
	}
	return (*r)[0].(float64)
}

func (cc *calcClient) Multiply(op1 float64, op2 float64) float64 {
	r,e:=cc.c.Call("Calc.Multiply",op1,op2)
	if(e!=nil) {
		cc.c.HandleError("Calc.Substract",e)
	}
	return (*r)[0].(float64)
}

func (cc *calcClient) Divide(op1 float64, op2 float64) (float64, os.Error) {
	r,e:=cc.c.Call("Calc.Divide",op1,op2)
	return (*r)[0].(float64),e
}

func TestClientServerLocal(t *testing.T) {
	s:=NewServer(&simplecalc{0},new(CalcServer))
	r1,w1:=io.Pipe()
	r2,w2:=io.Pipe()
	s.ServePipe(IO(r2,w1))
	c:=NewClient(IO(r1,w2))
	cc:=&calcClient{c}
	cc.Add(1,2)
}
