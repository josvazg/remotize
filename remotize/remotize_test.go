package remotize

import (
	"fmt"
	"io"
	"math"
	"os"
	"rpc"
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
	Server
}

// The Server wiring
func (s *CalcSrv) RPCAdd(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.impl.(Calc).Add((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCAddTo(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	a0 := (a.A[0]).(float64)
	r.R[0] = &a0
	s.impl.(Calc).AddTo((r.R[0]).(*float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCSubtract(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.impl.(Calc).Subtract((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCMultiply(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.impl.(Calc).Multiply((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcSrv) RPCDivide(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 2)
	r.R[0], r.R[1] = s.impl.(Calc).Divide((a.A[0]).(float64), (a.A[1]).(float64))
	if r.R[1] == nil {
		return nil
	}
	return (r.R[1]).(os.Error)
}

func (s *CalcSrv) RPCPi(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.impl.(Calc).Pi()
	return nil
}

func (s *CalcSrv) RPCRandomize(a *Args, r *Results) os.Error {
	s.impl.(Calc).Randomize()
	return nil
}

func (s *CalcSrv) RPCRandomizeSeed(a *Args, r *Results) os.Error {
	s.impl.(Calc).RandomizeSeed((a.A[0]).(float64))
	return nil
}

// The Client reference wiring
type CalcClt struct {
	Client
}

func (c *CalcClt) Add(op1 float64, op2 float64) float64 {
	r, e := Call(c.ToClient(), "CalcSrv.RPCAdd", op1, op2)
	if e != nil {
		HandleError(c.ToClient(), "Calc.Add", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcClt) AddTo(op1 *float64, op2 float64) {
	r, e := Call(c.ToClient(), "CalcSrv.RPCAddTo", op1, op2)
	if e != nil {
		HandleError(c.ToClient(), "Calc.Add", e)
	}
	*op1 = (r.R[0]).(float64)
}

func (c *CalcClt) Subtract(op1 float64, op2 float64) float64 {
	r, e := Call(c.ToClient(), "CalcSrv.RPCSubtract", op1, op2)
	if e != nil {
		HandleError(c.ToClient(), "Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcClt) Multiply(op1 float64, op2 float64) float64 {
	r, e := Call(c.ToClient(), "CalcSrv.RPCMultiply", op1, op2)
	if e != nil {
		HandleError(c.ToClient(), "Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcClt) Divide(op1 float64, op2 float64) (float64, os.Error) {
	r, e := Call(c.ToClient(), "CalcSrv.RPCDivide", op1, op2)
	return (r.R[0]).(float64), e
}

func (c *CalcClt) Pi() float64 {
	r, e := Call(c.ToClient(), "CalcSrv.RPCPi")
	if e != nil {
		HandleError(c.ToClient(), "Calc.Pi", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcClt) Randomize() {
	_, e := Call(c.ToClient(), "CalcSrv.RPCRandomize")
	if e != nil {
		HandleError(c.ToClient(), "Calc.Randomize", e)
	}
}

func (c *CalcClt) RandomizeSeed(seed float64) {
	_, e := Call(c.ToClient(), "CalcSrv.RPCRandomizeSeed", seed)
	if e != nil {
		HandleError(c.ToClient(), "Calc.RandomizeSeed", e)
	}
}

func TestClientServerLocal(t *testing.T) {
	fmt.Println("Test with reference code")
	s := &CalcSrv{}
	SetupServer(s.ToServer(), s, rpc.NewServer(), &simplecalc{})
	fmt.Println("s=", s)
	fmt.Println("register...")
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	go s.server.ServeConn(IO(r2, w1))

	c := &CalcClt{}
	SetupClient(c.ToClient(), rpc.NewClient(IO(r1, w2)))
	c.Randomize()
	fmt.Println("Randomize()")
	c.RandomizeSeed(4123423.2314)
	fmt.Println("RandomizeSeed(4123423.2314)")
	fmt.Println("1+2=", c.Add(1, 2))
	val := 1.0
	op := &val
	c.AddTo(op, 2)
	fmt.Println("AddTo 1+2=", *op)
	fmt.Println("1-2=", c.Subtract(1, 2))
	fmt.Println("1123.1234*-2.21432=", c.Multiply(1123.1234, -2.21432))
	d, e := c.Divide(1123.1234, -24.21432)
	fmt.Println("1123.1234/-2.21432=", d, " e=", e)
	fmt.Println("pi=", c.Pi())
}

func TestRemotize(t *testing.T) {
	Remotize(new(Calc))
}

func TestRemotized(t *testing.T) {
	fmt.Println("Test with autogenerated code")
	s := NewServer(new(Calc), &simplecalc{}, "")
	if s == nil {
		fmt.Println("Autogenerated code not ready yet")
		return
	}
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	go s.server.ServeConn(IO(r2, w1))
	ref := NewClient(rpc.NewClient(IO(r1, w2)), new(Calc), "")
	if ref == nil {
		fmt.Println("Autogenerated code client not ready yet")
		return
	}
	c := (interface{})(ref).(*CalcClt)
	c.Randomize()
	fmt.Println("Randomize()")
	c.RandomizeSeed(4123423.2314)
	fmt.Println("RandomizeSeed(4123423.2314)")
	fmt.Println("1+2=", c.Add(1, 2))
	// Support for in/out parameters
	val := 1.0
	op := &val
	c.AddTo(op, 2)
	fmt.Println("AddTo 1+2=", *op)
	fmt.Println("1-2=", c.Subtract(1, 2))
	fmt.Println("1123.1234*-2.21432=", c.Multiply(1123.1234, -2.21432))
	d, e := c.Divide(1123.1234, -24.21432)
	fmt.Println("1123.1234/-2.21432=", d, " e=", e)
	fmt.Println("pi=", c.Pi())
}

