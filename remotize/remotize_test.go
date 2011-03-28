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

// RPC server exported interface
type CalcRpcs struct {
	s *CalcSrv
}

// The server reference-wrapping type
type CalcSrv struct {
	ServerBase
	Rpcs *CalcRpcs
}

// The Server wiring
func (s *CalcSrv) Bind(server *rpc.Server, impl interface{}) {
	s.Base().Bind(server, impl)
	s.Rpcs = &CalcRpcs{s}
	server.Register(s.Rpcs)
}

func (s *CalcRpcs) Add(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.impl.(Calc).Add((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcRpcs) AddTo(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	a0 := (a.A[0]).(float64)
	r.R[0] = &a0
	s.s.impl.(Calc).AddTo((r.R[0]).(*float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcRpcs) Subtract(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.impl.(Calc).Subtract((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcRpcs) Multiply(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.impl.(Calc).Multiply((a.A[0]).(float64), (a.A[1]).(float64))
	return nil
}

func (s *CalcRpcs) Divide(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 2)
	r.R[0], r.R[1] = s.s.impl.(Calc).Divide((a.A[0]).(float64), (a.A[1]).(float64))
	if r.R[1] == nil {
		return nil
	}
	return (r.R[1]).(os.Error)
}

func (s *CalcRpcs) Pi(a *Args, r *Results) os.Error {
	r.R = make([]interface{}, 1)
	r.R[0] = s.s.impl.(Calc).Pi()
	return nil
}

func (s *CalcRpcs) Randomize(a *Args, r *Results) os.Error {
	s.s.impl.(Calc).Randomize()
	return nil
}

func (s *CalcRpcs) RandomizeSeed(a *Args, r *Results) os.Error {
	s.s.impl.(Calc).RandomizeSeed((a.A[0]).(float64))
	return nil
}

// Local Remote Interface reference
type CalcRemote struct {
	c *CalcClt
}

// The Client reference wiring
type CalcClt struct {
	ClientBase
}

// Bind client
func (c *CalcClt) Bind(client *rpc.Client) {
	c.Base().Bind(client)
	c.remote = &CalcRemote{c}
}


func (c *CalcRemote) Add(op1 float64, op2 float64) float64 {
	r, e := Call(c.c.Base(), "CalcRpcs.Add", op1, op2)
	if e != nil {
		HandleError(c.c.Base(), "Calc.Add", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcRemote) AddTo(op1 *float64, op2 float64) {
	r, e := Call(c.c.Base(), "CalcRpcs.AddTo", op1, op2)
	if e != nil {
		HandleError(c.c.Base(), "Calc.Add", e)
	}
	*op1 = (r.R[0]).(float64)
}

func (c *CalcRemote) Subtract(op1 float64, op2 float64) float64 {
	r, e := Call(c.c.Base(), "CalcRpcs.Subtract", op1, op2)
	if e != nil {
		HandleError(c.c.Base(), "Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcRemote) Multiply(op1 float64, op2 float64) float64 {
	r, e := Call(c.c.Base(), "CalcRpcs.Multiply", op1, op2)
	if e != nil {
		HandleError(c.c.Base(), "Calc.Substract", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcRemote) Divide(op1 float64, op2 float64) (float64, os.Error) {
	r, e := Call(c.c.Base(), "CalcRpcs.Divide", op1, op2)
	return (r.R[0]).(float64), e
}

func (c *CalcRemote) Pi() float64 {
	r, e := Call(c.c.Base(), "CalcRpcs.Pi")
	if e != nil {
		HandleError(c.c.Base(), "Calc.Pi", e)
	}
	return (r.R[0]).(float64)
}

func (c *CalcRemote) Randomize() {
	_, e := Call(c.c.Base(), "CalcRpcs.Randomize")
	if e != nil {
		HandleError(c.c.Base(), "Calc.Randomize", e)
	}
}

func (c *CalcRemote) RandomizeSeed(seed float64) {
	_, e := Call(c.c.Base(), "CalcRpcs.RandomizeSeed", seed)
	if e != nil {
		HandleError(c.c.Base(), "Calc.RandomizeSeed", e)
	}
}

func TestClientServerLocal(t *testing.T) {
	fmt.Println("Test with reference code")
	server := rpc.NewServer()
	srv := &CalcSrv{}
	srv.Bind(server, &simplecalc{})
	s := (interface{})(srv).(Server)

	fmt.Println("s=", s)
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	go server.ServeConn(IO(r2, w1))

	client := rpc.NewClient(IO(r1, w2))
	clt := &CalcClt{}
	clt.Bind(client)
	c := clt.Remote().(Calc)
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
	server := rpc.NewServer()
	s := NewServer(server, new(Calc), &simplecalc{}, "")
	if s == nil {
		fmt.Println("Autogenerated code not ready yet")
		return
	}
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	go server.ServeConn(IO(r2, w1))
	ref := NewClient(rpc.NewClient(IO(r1, w2)), new(Calc), "")
	if ref == nil {
		fmt.Println("Autogenerated code client not ready yet")
		return
	}
	c := (ref).Remote().(Calc)
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

