// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// The mapper package implements a extended map that can be thread safe (or not)
// , insertion ordered (or not) and typed (or not)
package mapper

import (
	"fmt"
	"reflect"
)

const ( 
	put = iota 
	get 
	remove 
	str 
	keys
	clear 
) 

// The Mapper interface is the one provided for extended Maps created by this 
// package.
//
// Other implementations might use this interface as well
type Mapper interface {
	Put(key string, value interface{})
	Get(key string) (value interface{}, ok bool)
	Remove(key string) (value interface{}, ok bool)
	Keys() []string 
	Clear()
}

type msg struct { 
	op	int 
	ok	bool
	str	string 
	value	interface{}
	keys	[]string
	reply	chan msg 
} 

// hidden mapper implementation
type themap struct { 
        values	map[string]interface{} 
	typed	reflect.Type
        ch	chan msg 
	keys	[]string
	order	map[string]int
}

// NewMapper creates a preconfigured Mapper implementation:
//
// "threadSafe" and "linked" control whether the Mapper is thread safe and insertion
// -ordered or not respectively
//
// "typed" is a type or a sample value (or pointer to value) of the type to force
// the map to be. Or it can be nil if the Mapper is NOT typed
func NewMapper(threadSafe bool, linked bool, typed interface{}) Mapper { 
        m:=new(themap); 
        m.values=make(map[string]interface{}) 
	if(threadSafe) {
	        m.ch=make(chan msg) 
	        go m.handler() 
	}
	if(linked) {
		m.keys=make([]string,10)
		m.order=make(map[string]int)
	}
	if(typed!=nil) {
	        isAType:=false
		m.typed,isAType=typed.(reflect.Type)
		if(!isAType) {
			m.typed=reflect.Typeof(typed)
		}
	}
        return m; 
} 

func (m *themap) handler() { 
        for{ 
                r:=<-m.ch 
                switch r.op {
                        case put: 
                                m.put(r.str,r.value)
                        case get: 
                                r.value,r.ok=m.get(r.str)
                        case remove: 
                                r.value,r.ok=m.remove(r.str)
			case str: 
                                r.str=m.tostring() 
			case keys:
				r.keys=m.listKeys()
                        case clear: 
                                m.values=make(map[string]interface{}) 
				if(m.keys!=nil) {
					m.keys=make([]string,10)
				}
                } 
		if(r.reply!=nil) {
			r.reply<-r
		}
        } 
} 

func (m *themap) put(key string, value interface{}) {
	m.values[key]=value
	if(m.keys!=nil) {
		m.keys=append(m.keys,key)
		m.order[key]=len(m.keys)-1
	}
}

func (m *themap) Put(key string, value interface{}) { 
	if(m.ch!=nil) {
	        m.ch<-msg{put,false,key,value,nil,nil} 		
	} else {
		m.put(key,value)
	}
} 

func (m *themap) get(key string) (interface{}, bool) {
	value,ok:=m.values[key]
	return value,ok
}

func (m *themap) Get(key string) (interface{}, bool) { 
	if(m.ch!=nil) {
	        rch:=make(chan msg) 
        	m.ch<-msg{get,false,key,nil,nil,rch} 
	        r:=<-rch 
        	return r.value, r.ok
	}
	return m.get(key)
} 

func (m *themap) remove(key string) (value interface{},ok bool) {
	value,ok=m.get(key)
	if(ok) {
		m.values[key]=nil,false 
	}
	if(m.keys!=nil) {
		i:=m.order[key]
		before:=m.keys[:i]
		after:=m.keys[i+1:]
		m.keys=append(before,after...)
	}
	return
}	

func (m *themap) Remove(key string) (interface{},bool) { 
	if(m.ch!=nil) {
	        rch:=make(chan msg) 
        	m.ch<-msg{remove,false,key,nil,nil,rch} 
	        r:=<-rch 
        	return r.value,r.ok
	}
	return m.remove(key)
} 

func (m *themap) listKeys() ([]string) {
	if(m.keys!=nil) {
		return m.keys
	}
	r:=make([]string,0,10)
	for k,_:=range(m.values) {
		r=append(r,k)
	}
	return r
}

func (m *themap) Keys() ([]string) {
	if(m.ch!=nil) {
	        rch:=make(chan msg) 
        	m.ch<-msg{keys,false,"",nil,nil,rch} 
	        r:=<-rch
        	return r.keys
	}
	return m.listKeys()
}

func (m *themap) Clear() {
	if(m.ch!=nil) {
        	m.ch<-msg{clear,false,"",nil,nil,nil}
	} else {
		m.values=make(map[string]interface{}) 
		if(m.keys!=nil) {
			m.keys=make([]string,10)
		}
	}
}

func (m *themap) tostring() string {
	ts:="ThreadUnsafe"
	if(m.ch!=nil) {
	  ts="ThreadSafe"
	}
	ln:="Unlinked"
	if(m.keys!=nil) {
	  ln="Linked"
	}
	ty:=""
	if(m.typed!=nil) {
	  ty=fmt.Sprintf("<%v>",m.typed)
	}
	return fmt.Sprintf("%v%v%v%v",ts,ln,ty,m.values)
}

func (m *themap) String() string {
	if(m.ch!=nil) {
	        rch:=make(chan msg) 
        	m.ch<-msg{str,false,"",nil,nil,rch} 
	        r:=<-rch 
        	return r.str 
	}
	return m.tostring()
}


