package remotize

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"testing"
)

// The interface
type Calc interface {
	Add(float64, float64) float64
	AddTo(*float64, float64)
	Subtract(float64, float64) float64
	Multiply(float64, float64) float64
	Divide(float64, float64) (float64, os.Error)
	Pi() float64
	Randomize()
	RandomizeSeed(float64)
}

// The type implementing it
type simplecalc struct{}

func (sc *simplecalc) Add(op1 float64, op2 float64) float64 {
	return op1 + op2
}

func (sc *simplecalc) AddTo(op1 *float64, op2 float64) {
	*op1 = *op1 + op2
}

func (sc *simplecalc) Subtract(op1 float64, op2 float64) float64 {
	return op1 - op2
}

func (sc *simplecalc) Multiply(op1 float64, op2 float64) float64 {
	return op1 * op2
}

func (sc *simplecalc) Divide(op1 float64, op2 float64) (float64, os.Error) {
	if op2 == 0 {
		return 0, os.NewError("Divide " + strconv.Ftoa64(op1, 'f', -1) + " by ZERO!?!")
	}
	return op1 / op2, nil
}

func (sc *simplecalc) Pi() float64 {
	return math.Pi
}

func (sc *simplecalc) Randomize() {
	fmt.Println("Randomized!")
}

func (sc *simplecalc) RandomizeSeed(seed float64) {
	fmt.Println(seed, "randomized!")
}

// The server reference-wrapping type
type CalcSrv struct {
	s Calc
}

// The Server wiring
func (s *CalcSrv) RPCAdd(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.Add((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCAddTo(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	a0 := (a.A[0]).(float64)
	r.R[0] = &a0
	s.s.AddTo((r.R[0]).(*float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCSubtract(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.Subtract((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCMultiply(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.Multiply((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCDivide(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 2)
	r.R[0], r.R[1] = s.s.Divide((a.A[0]).(float64), (a.A[1]).(float64))
	if r.R[1] == nil {
		return nil
	}
	return (r.R[1]).(os.Error)
}

func (s *CalcSrv) RPCPi(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.Pi()
	return nil
}

func (s *CalcSrv) RPCRandomize(a *Args, r *Results) os.Error {
	s.s.Randomize()
	return nil
}

func (s *CalcSrv) RPCRandomizeSeed(a *Args, r *Results) os.Error {
	s.s.RandomizeSeed((a.A[0]).(float64))
	return nil
}

// The Client reference wiring
type calcClient struct {
	c *Client
}

func (cc *calcClient) Add(op1 float64, op2 float64) float64 {
	r, e := cc.c.Call("CalcSrv.RPCAdd", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Add", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) AddTo(op1 *float64, op2 float64) {
	r, e := cc.c.Call("CalcSrv.RPCAddTo", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Add", e)
	}
	*op1 = (r.R[0]).(float64)
}

func (cc *calcClient) Subtract(op1 float64, op2 float64) float64 {
	r, e := cc.c.Call("CalcSrv.RPCSubtract", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Multiply(op1 float64, op2 float64) float64 {
	r, e := cc.c.Call("CalcSrv.RPCMultiply", op1, op2)
	if e != nil {
		cc.c.HandleError("Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Divide(op1 float64, op2 float64) (float64, os.Error) {
	r, e := cc.c.Call("CalcSrv.RPCDivide", op1, op2)
	return (r.R[0]).(float64), e
}

func (cc *calcClient) Pi() float64 {
	r, e := cc.c.Call("CalcSrv.RPCPi")
	if e != nil {
		cc.c.HandleError("Calc.Pi", e)
	}
	return (r.R[0]).(float64)
}

func (cc *calcClient) Randomize() {
	_, e := cc.c.Call("CalcSrv.RPCRandomize")
	if e != nil {
		cc.c.HandleError("Calc.Randomize", e)
	}
}

func (cc *calcClient) RandomizeSeed(seed float64) {
	_, e := cc.c.Call("CalcSrv.RPCRandomizeSeed", seed)
	if e != nil {
		cc.c.HandleError("Calc.RandomizeSeed", e)
	}
}

func TestClientServerLocal(t *testing.T) {
	fmt.Println("Test with reference code")
	s := NewServer(&CalcSrv{&simplecalc{}})
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
	val := 1.0
	op := &val
	cc.AddTo(op, 2)
	fmt.Println("AddTo 1+2=", *op)
	fmt.Println("1-2=", cc.Subtract(1, 2))
	fmt.Println("1123.1234*-2.21432=", cc.Multiply(1123.1234, -2.21432))
	d, e := cc.Divide(1123.1234, -24.21432)
	fmt.Println("1123.1234/-2.21432=", d, " e=", e)
	fmt.Println("pi=", cc.Pi())
}

func TestRemotize(t *testing.T) {
	Remotize(new(Calc))
}
/*
func TestRemotized(t *testing.T) {
	fmt.Println("Test with autogenerated code")
	s := NewServer(&CalcServer{&simplecalc{}})
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	s.ServePipe(IO(r2, w1))
	c := NewClient(IO(r1, w2))
	cc := &CalcClient{c}
	cc.Randomize()
	fmt.Println("Randomize()")
	cc.RandomizeSeed(4123423.2314)
	fmt.Println("RandomizeSeed(4123423.2314)")
	fmt.Println("1+2=", cc.Add(1, 2))
	// Support for in/out parameters
	//val := 1.0 
	//op := &val
	//cc.AddTo(op, 2)
	//fmt.Println("AddTo 1+2=", *op)
	fmt.Println("1-2=", cc.Subtract(1, 2))
	fmt.Println("1123.1234*-2.21432=", cc.Multiply(1123.1234, -2.21432))
	d, e := cc.Divide(1123.1234, -24.21432)
	fmt.Println("1123.1234/-2.21432=", d, " e=", e)
	fmt.Println("pi=", cc.Pi())
}*/

