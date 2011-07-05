package remotize

import (
	"fmt"
	"io"
	"rpc"
	test "testing"
)

func TestRemotizedCalc(t *test.T) {
	server := rpc.NewServer()
	r := NewRemote(server, new(Calc))
	if r == nil {
		fmt.Println("Autogenerated code not ready yet")
		return
	}
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	go server.ServeConn(IO(r2, w1))
	l := NewLocal(rpc.NewClient(IO(r1, w2)), new(Calc))
	if l == nil {
		fmt.Println("Autogenerated code client not ready yet")
		return
	}
	c := (l).(Calcer)
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

