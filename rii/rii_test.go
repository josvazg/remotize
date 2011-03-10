package rii

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"
)

// The interface
type Calc interface {
	Add(float64, float64) float64
	Subtract(float64, float64) float64
	Multiply(float64, float64) float64
	Divide(float64, float64) (float64, os.Error)
	Pi() float64
	Randomize()
	RandomizeSeed(float64)
}

// The type implementing it
type simplecalc struct {
	r float64
}

func (sc *simplecalc) Add(op1 float64, op2 float64) float64 {
	sc.r = op1 + op2
	return sc.r
}

func (sc *simplecalc) Subtract(op1 float64, op2 float64) float64 {
	sc.r = op1 - op2
	return sc.r
}

func (sc *simplecalc) Multiply(op1 float64, op2 float64) float64 {
	sc.r = op1 * op2
	return sc.r
}

func (sc *simplecalc) Divide(op1 float64, op2 float64) (float64, os.Error) {
	if op2 == 0 {
		return 0, os.NewError("Divide " + strconv.Ftoa64(op1, 'f', -1) + " by ZERO!?!")
	}
	sc.r = op1 / op2
	return sc.r, nil
}

func (sc *simplecalc) Pi() float64 {
	return 3.14159265
}

func (sc *simplecalc) Randomize() {
	fmt.Println("Randomized!")
}

func (sc *simplecalc) RandomizeSeed(seed float64) {
	fmt.Println(seed, "randomized!")
}

// The server reference-wrapping type
type CalcSrv struct {
	i simplecalc
}

// The Server wiring
func (s *CalcSrv) Add(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.i.Add((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) Subtract(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.i.Subtract((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) Multiply(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.i.Multiply((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) Divide(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 2)
	r.R[0], r.R[1] = s.i.Divide((a.A[0]).(float64), (a.A[1]).(float64))
	if r.R[1] == nil {
		return nil
	}
	return (r.R[1]).(os.Error)
}

func (s *CalcSrv) Pi(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.i.Pi()
	return nil
}

func (s *CalcSrv) Randomize(a *Args, r *Results) os.Error {
	s.i.Randomize()
	return nil
}

func (s *CalcSrv) RandomizeSeed(a *Args, r *Results) os.Error {
	s.i.RandomizeSeed((a.A[0]).(float64))
	return nil
}

// The Client reference wiring
type calcClient struct {
	c *Client
}

func (cc *calcClient) Add(op1 float64, op2 float64) float64 {
	r, e := cc.c.Call("CalcSrv.Add", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Add", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Subtract(op1 float64, op2 float64) float64 {
	r, e := cc.c.Call("CalcSrv.Subtract", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Multiply(op1 float64, op2 float64) float64 {
	r, e := cc.c.Call("CalcSrv.Multiply", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Divide(op1 float64, op2 float64) (float64, os.Error) {
	r, e := cc.c.Call("CalcSrv.Divide", op1, op2)
	return (r.R[0]).(float64), e
}

func (cc *calcClient) Pi() float64 {
	r, e := cc.c.Call("CalcSrv.Pi")
	if e != nil {
		cc.c.HandleError("Calc.Pi", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Randomize() {
	_, e := cc.c.Call("CalcSrv.Randomize")
	if e != nil {
		cc.c.HandleError("Calc.Randomize", e)
	}
}

func (cc *calcClient) RandomizeSeed(seed float64) {
	_, e := cc.c.Call("CalcSrv.RandomizeSeed", seed)
	if e != nil {
		cc.c.HandleError("Calc.RandomizeSeed", e)
	}
}

func TestClientServerLocal(t *testing.T) {
	s := NewServer(new(CalcSrv))
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	s.ServePipe(IO(r2, w1))
	c := NewClient(IO(r1, w2))
	cc := &calcClient{c}
	cc.Randomize()
	fmt.Println("Randomize()")
	cc.RandomizeSeed(4123423.2314)
	fmt.Println("RandomizeSeed(4123423.2314)")
	fmt.Println("1+2=", cc.Add(1, 2))
	fmt.Println("1-2=", cc.Subtract(1, 2))
	fmt.Println("1123.1234*-2.21432=", cc.Multiply(1123.1234, -2.21432))
	d, e := cc.Divide(1123.1234, -24.21432)
	fmt.Println("1123.1234/-2.21432=", d, " e=", e)
	fmt.Println("pi=", cc.Pi())
}

func TestRemotize(t *testing.T) {
	Remotize(new(Calc))
}

