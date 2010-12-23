// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// rii package is the Remote Interface Invocation foundation allowing go 
// programs to use out-of-process services defined by an interface, either
// locally or remotelly without worring (too much or too soon) 
// about the communications. With this package you can remotize local parts of 
// the program or load them dynamically as a plugin.
package rii

import (
	"fmt"
	"reflect"
)

func Remotize(iface interface{}) {
	if it,ok:=iface.(*reflect.InterfaceType);ok {
		remotize(it)
		return
	}
	t:=reflect.Typeof(iface)
	if pt,ok:=t.(*reflect.PtrType);ok {
		//fmt.Println("Remotizing",pt,"->",pt.Elem(),"...")
		if it,ok2:=pt.Elem().(*reflect.InterfaceType);ok2 {
			remotize(it)
			return
		}
	} 
	fmt.Println("Can't remotize",iface,"of non interface type",t)
}

func remotize(it *reflect.InterfaceType) {
	fmt.Println("Remotizing interface",it)
	nm:=it.NumMethod()
	fmt.Println("Interface exports ",nm,"methods")
	for i:=0;i<nm;i++ {
		m:=it.Method(i)
		fmt.Println("Method ",m)
		analyze(m)
	}
}

type rmethodspec struct {
	name	string
	f		*reflect.FuncType
	in		[]reflect.Type
	inout	[]reflect.Type
	out		[]reflect.Type
}

func analyze(m reflect.Method) *rmethodspec {
	var rm rmethodspec
	rm.name=m.Name
	rm.f=m.Type
	nin:=rm.f.NumIn()
	if(nin>0) {
		for i:=0;i<nin;i++ {
			ta:=rm.f.In(i)
			if isInOut(ta) {
				rm.inout=append(rm.in,ta)
			} else {
				rm.in=append(rm.in,ta)
			}			
		}
		
	}
	nout:=rm.f.NumOut()
	if(nout>0) {
		for i:=0;i<nout;i++ {
			tr:=rm.f.Out(i)
			rm.out=append(rm.out,tr)			
		}
	}
	fmt.Println("Method",rm.name,"IN:",rm.in,"IN/OUT:",rm.inout,"OUT:",rm.out)
	return &rm
}

func isInOut(t reflect.Type) bool {
	switch t.Kind() {
		case reflect.Ptr:
			return true
		case reflect.Slice:
			return true
	}
	return false
}


