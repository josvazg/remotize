package mapper

import (
	"fmt"
	"net"
	"os"
	"testing"
)

func TestMap(t *testing.T) {
	m := NewMapper(false, false, nil)
	fmt.Println("Test ", m)
	m.Put("a", "texto de a")
	m.Put("b", 2)
	m.Put("c", 1234)
	fmt.Println("m=", m)
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		fmt.Println(k+":", v)
	}
}

func TestThreadSafeMap(t *testing.T) {
	m := NewMapper(true, false, nil)
	quitChan := make(chan int)
	fmt.Println("Test ", m)
	go func(m Mapper, ch chan int) {
		m.Put("1", 1)
		fmt.Println("Test One ", m)
		ch <- 1
	}(m, quitChan)
	go func(m Mapper, ch chan int) {
		m.Put("1", 2)
		fmt.Println("Test Two ", m)
		ch <- 1
	}(m, quitChan)
	<-quitChan
	<-quitChan
}

type MCast struct {
	c *net.UDPConn
	a *net.UDPAddr
}

func TestMCast(t *testing.T) {
	mcaddr := "224.0.0.254:18081"
	mc1, e := JoinMulticast(mcaddr)
	if e != nil {
		t.Fatal(e)
	}
	go func(mc *MCast) {
		msg := "Hi multicast!"
		n, e := mc.Write(([]byte)(msg))
		if e != nil {
			t.Fatal(e)
		}
		fmt.Println("<-", n, "bytes:", msg)
		e = mc.LeaveMulticast()
		if e != nil {
			t.Fatal(e)
		}
	}(mc1)
	mc2, e := JoinMulticast(mcaddr)
	if e != nil {
		t.Fatal(e)
	}
	b := make([]byte, 1500)
	n, e := mc2.Read(b)
	if e != nil {
		t.Fatal(e)
	}
	if n > 0 {
		fmt.Println(n, "bytes:", string(b), "<-")
	} else {
		fmt.Println("got 0bytes:")
	}
	e = mc2.LeaveMulticast()
	if e != nil {
		t.Fatal(e)
	}
}

func JoinMulticast(a string) (*MCast, os.Error) {
	ua, e := net.ResolveUDPAddr(a)
	if e != nil {
		return nil, e
	}
	addr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: ua.Port,
	}
	c, e := net.ListenUDP("udp4", addr)
	if e != nil {
		return nil, e
	}
	e = c.JoinGroup(ua.IP)
	if e != nil {
		return nil, e
	}
	fmt.Println("Joined to multicast IP", ua.IP, "on port", ua.Port)
	return &MCast{c, ua}, nil
}

func (mc *MCast) Write(b []byte) (int, os.Error) {
	return mc.c.WriteTo(b, mc.a)
}

func (mc *MCast) Read(b []byte) (int, os.Error) {
	return mc.c.Read(b)
}

func (mc *MCast) LeaveMulticast() os.Error {
	e := mc.c.LeaveGroup(mc.a.IP)
	if e != nil {
		return e
	}
	return mc.c.Close()
}

