package sample

import (
	"http"
	"net"
	"os"
	"github.com/josvazg/remotize"
	"rpc"
	test "testing"
)

func dieOnError(t *test.T, e os.Error) {
	if e != nil {
		t.Fatalf("listen error: %v", e)
	}
}

const (
	Add = iota
	AddTo
	Subtract
	Multiply
	Divide
	Pi
	Randomize
	RandomizeSeed
)

type OpType int

var calcTests = []struct {
	op  OpType
	arg []float64
}{
	{Add, []float64{-.5, 2.7}},
	{AddTo, []float64{782436578.02342, 27367423.423}},
	{Subtract, []float64{812345.24235, 12342.346}},
	{Multiply, []float64{23874.12342, 42314.23484}},
	{Divide, []float64{892347.587, 23432.234}},
	{Pi, nil},
	{Randomize, nil},
	{RandomizeSeed, []float64{7234643.21432}},
}

func startCalcerServer(t *test.T) string {
	// You can access the remotized code directly, it should be created by now...
	r := NewCalcerService(new(Calc))
	rpc.Register(r)
	rpc.HandleHTTP()
	addr := ":1234"
	l, e := net.Listen("tcp", addr)
	dieOnError(t, e)
	go http.Serve(l, nil)
	return "localhost" + addr
}

func getRemoteCalcerRef(t *test.T, saddr string) Calcer {
	client, e := rpc.DialHTTP("tcp", saddr)
	dieOnError(t, e)
	return NewRemoteCalcer(client)
}

func check(t *test.T, i interface{}, ok bool) {
	if !ok {
		t.Fatalf("Error in ", i)
	}
}

func TestRemotizedCalc(t *test.T) {
	serveraddr := startCalcerServer(t)
	calc := new(Calc)
	rcalc := getRemoteCalcerRef(t, serveraddr)
	for _, ct := range calcTests {
		switch ct.op {
		case Add:
			check(t, ct, calc.Add(ct.arg[0], ct.arg[1]) == rcalc.Add(ct.arg[0], ct.arg[1]))
		case AddTo:
			add := ct.arg[0]
			radd := add
			calc.AddTo(&add, ct.arg[1])
			rcalc.AddTo(&radd, ct.arg[1])
			check(t, ct, add == radd)
		case Subtract:
			check(t, ct,
				calc.Subtract(ct.arg[0], ct.arg[1]) == rcalc.Subtract(ct.arg[0], ct.arg[1]))
		case Multiply:
			check(t, ct,
				calc.Multiply(ct.arg[0], ct.arg[1]) == rcalc.Multiply(ct.arg[0], ct.arg[1]))
		case Divide:
			d1, e := calc.Divide(ct.arg[0], ct.arg[1])
			dieOnError(t, e)
			d2, e := rcalc.Divide(ct.arg[0], ct.arg[1])
			dieOnError(t, e)
			check(t, ct, d1 == d2)
		case Pi:
			check(t, ct, calc.Pi() == rcalc.Pi())
		case Randomize:
			rcalc.Randomize()
		case RandomizeSeed:
			rcalc.RandomizeSeed(ct.arg[0])
		}
	}
}

var ustorerTests = []struct {
	shorturl, url string
}{
	{"ib", "www.ibm.com"},
	{"gg", "www.google.com"},
	{"m$", "www.microsoft.com"},
	{"ap", "www.apple.com"},
}

func startStorerServer(t *test.T, us URLStorer) string {
	// You can also search the service by passing the impleemntation to remotize...
	r := remotize.NewService(us)
	rpc.Register(r)
	rpc.HandleHTTP()
	addr := ":12345"
	l, e := net.Listen("tcp", addr)
	dieOnError(t, e)
	go http.Serve(l, nil)
	return "localhost" + addr
}

func getRemoteStorerRef(t *test.T, saddr string) URLStorer {
	client, e := rpc.DialHTTP("tcp", saddr)
	dieOnError(t, e)
	return remotize.NewRemote(client, new(URLStorer)).(URLStorer)
}

func TestRemotizedURLStorer(t *test.T) {
	serveraddr := startStorerServer(t, NewURLStore())
	us := NewURLStore()
	rus := getRemoteStorerRef(t, serveraddr)
	for _, tu := range ustorerTests {
		us.Set(tu.shorturl, tu.url)
		rus.Set(tu.shorturl, tu.url)
		check(t, tu, us.Get(tu.shorturl) == rus.Get(tu.shorturl))
	}
}
