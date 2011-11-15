package sample

import (
	"os"
	test "testing"
)

func dieOnError(t *test.T, e os.Error) {
	if e != nil {
		t.Fatalf("listen error: %v", e)
	}
}

func check(t *test.T, i interface{}, ok bool) {
	if !ok {
		t.Fatalf("Error in ", i)
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

func TestRemotizedCalc(t *test.T) {
	serveraddr, e := startCalcerServer()
	dieOnError(t, e)
	calc := new(Calc)
	rcalc, e := getRemoteCalcerRef(serveraddr)
	dieOnError(t, e)
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

func TestRemotizedURLStorer(t *test.T) {
	serveraddr, e := startStorerServer(NewURLStore())
	dieOnError(t, e)
	us := NewURLStore()
	rus, e := getRemoteStorerRef(serveraddr)
	dieOnError(t, e)
	for _, tu := range ustorerTests {
		us.Set(tu.shorturl, tu.url)
		rus.Set(tu.shorturl, tu.url)
		check(t, tu, us.Get(tu.shorturl) == rus.Get(tu.shorturl))
	}
}
