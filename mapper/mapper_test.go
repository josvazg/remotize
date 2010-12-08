package mapper

import (
	"testing"
	"fmt"
)

func TestMap(t *testing.T) {
	m:=NewMapper(false,false,nil)
	fmt.Println("Test ",m)
	m.Put("a","texto de a")
	m.Put("b",2)
	m.Put("c",1234)
	fmt.Println("m=",m)
	for _,k:=range(m.Keys()) {
		v,_:=m.Get(k)
		fmt.Println(k+":",v)
	}		
}

func TestThreadSafeMap(t *testing.T) {
	m:=NewMapper(true,false,nil)
	quitChan:=make(chan int)
	fmt.Println("Test ",m)
	go func(m Mapper, ch chan int) {
		m.Put("1",1)		
		fmt.Println("Test One ",m)
		ch<-1
	}(m,quitChan)
	go func(m Mapper, ch chan int) {
		m.Put("1",2)
		fmt.Println("Test Two ",m)
		ch<-1
	}(m,quitChan)
	<-quitChan
	<-quitChan
}

