package riisample

import (
	"rii"
	"io"
)

func f1(calc interface{}, args *[]interface{}) *[]interface{} {
	var rs []interface{}
	r,e:=calc.(Calc).add(((*args)[0]).(float),((*args)[1]).(float))
	rs=append(rs,r)
	rs=append(rs,e)
	return &rs
}

func f2(calc interface{}, args *[]interface{}) *[]interface{} {
	var rs []interface{}
	r,e:=calc.(Calc).subtract(((*args)[0]).(float),((*args)[1]).(float))
	rs=append(rs,r)
	rs=append(rs,e)
	return &rs
}

func f3(calc interface{}, args *[]interface{}) *[]interface{} {
	var rs []interface{}
	r,e:=calc.(Calc).multiply(((*args)[0]).(float),((*args)[1]).(float))
	rs=append(rs,r)
	rs=append(rs,e)
	return &rs
}

func f4(calc interface{}, args *[]interface{}) *[]interface{} {
	var rs []interface{}
	r,e:=calc.(Calc).divide(((*args)[0]).(float),((*args)[1]).(float))
	rs=append(rs,r)
	rs=append(rs,e)
	return &rs
}

func newCalcSkel(r io.ReadCloser, w io.Writer, calc Calc) *rii.Skel {
	cs:=rii.NewSkel(r,w,calc)
	cs.Add(f1)
	cs.Add(f2)
	cs.Add(f3)
	cs.Add(f4)
	return cs
}
