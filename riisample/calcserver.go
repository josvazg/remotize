package riisample

import (
	"rii"
	"os"
)

func newCalcServer(calc Calc) *rii.Server {
	cs:=rii.NewServer(calc)
	cs.Add(calc_add)
	cs.Add(calc_substract)
	cs.Add(calc_multiply)
	cs.Add(calc_divide)
	return cs
}

func calc_add(calc interface{}, args []interface{}, 
		res *[]interface{}) os.Error {
	r,e:=calc.(Calc).add((args[0]).(float),(args[1]).(float))
	*res=append(*res,r)
	*res=append(*res,e)
	return nil
}

func calc_substract(calc interface{}, args []interface{}, 
		res *[]interface{}) os.Error {
	r,e:=calc.(Calc).subtract((args[0]).(float),(args[1]).(float))
	*res=append(*res,r)
	*res=append(*res,e)
	return nil
}

func calc_multiply(calc interface{}, args []interface{}, 
		res *[]interface{}) os.Error {
	r,e:=calc.(Calc).multiply((args[0]).(float),(args[1]).(float))
	*res=append(*res,r)
	*res=append(*res,e)
	return nil
}

func calc_divide(calc interface{}, args []interface{}, 
		res *[]interface{}) os.Error {
	r,e:=calc.(Calc).divide((args[0]).(float),(args[1]).(float))
	*res=append(*res,r)
	*res=append(*res,e)
	return nil
}


