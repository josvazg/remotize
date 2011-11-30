package sample

import (
	"sample/dep"
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

const (
	Create = iota
	Mkdir
	Remove
	FileInfo
	Rename
	ReadAt
	WriteAt
	Readdir
)

var rfileTests = []struct {
	op            OpType
	file, newname string
	b             []byte
	off           int64
	n             int
}{
	{Mkdir, "somedir/subdir", "", nil, 0, 0},
	{Create, "nonexistingdir/somefile", "", nil, 0, 0},
	{Create, "somedir/somefile", "", nil, 0, 0},
	{FileInfo, "somedir/somefile", "", nil, 0, 0},
	{Rename, "somedir/somefile", "somedir/subdir/somefile.txt", nil, 0, 0},
	{WriteAt, "somedir/subdir/somefile.txt", "", []byte("Hello!"), 10, 0},
	{ReadAt, "somedir/subdir/somefile.txt", "", make([]byte, 10), 8, 0},
	{Readdir, "somedir/subdir", "", nil, 0, 10},
	{Remove, "somedir", "", nil, 0, 0},
}

func TestRemotizedFiler(t *test.T) {
	lprefix := "local/"
	rprefix := "remote/"
	serveraddr, e := startFilerServer()
	dieOnError(t, e)
	fs := new(dep.FileService)
	rfs, e := getRemoteFileServicerRef(serveraddr)
	dieOnError(t, e)
	for _, ft := range rfileTests {
		switch ft.op {
		case Create:
			check(t, ft, fs.Create(lprefix+ft.file) == rfs.Create(rprefix+ft.file))
		case Mkdir:
			check(t, ft, fs.Mkdir(lprefix+ft.file) == rfs.Mkdir(rprefix+ft.file))
		case Remove:
			check(t, ft, fs.Remove(lprefix+ft.file) == rfs.Remove(rprefix+ft.file))
		case FileInfo:
			lfi, le := fs.FileInfo(lprefix + ft.file)
			rfi, re := rfs.FileInfo(rprefix + ft.file)
			check(t, ft, le == re)
			check(t, ft, lfi.Size == rfi.Size)
			check(t, ft, lfi.Mode == rfi.Mode)
		case Rename:
			check(t, ft, fs.Rename(lprefix+ft.file,
				lprefix+ft.newname) == rfs.Rename(rprefix+ft.file, rprefix+ft.newname))
		case ReadAt:
			lr, le := fs.ReadAt(lprefix+ft.file, ft.b, ft.off)
			rr, re := rfs.ReadAt(rprefix+ft.file, ft.b, ft.off)
			check(t, ft, le == re)
			check(t, ft, lr == rr)
		case WriteAt:
			lr, le := fs.WriteAt(lprefix+ft.file, ft.b, ft.off)
			rr, re := rfs.WriteAt(rprefix+ft.file, ft.b, ft.off)
			check(t, ft, le == re)
			check(t, ft, lr == rr)
		case Readdir:
			lfi, le := fs.Readdir(lprefix+ft.file, ft.n)
			rfi, re := rfs.Readdir(rprefix+ft.file, ft.n)
			check(t, ft, le == re)
			check(t, ft, lfi != nil && rfi != nil && len(lfi) == len(rfi))
		}
	}
}

