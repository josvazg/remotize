package sample

import (
	"fmt"
	"http"
	"net"
	"os"
	"rpc"
	test "testing"
)

func dieOnError(t *test.T, e os.Error) {
	if e != nil {
		t.Fatal("listen error:", e)
	}
}

func fireUpCalcerServer(t *test.T) string {
	// You can access the remotized code directly, it should be created by now...
	r:=NewCalcerService(new(Calc))
	rpc.Register(r)
	rpc.HandleHTTP()
	addr:=":1234"
	l, e := net.Listen("tcp", addr)
	dieOnError(t,e)
	go http.Serve(l, nil)
	return "localhost"+addr
}

func getRemoteRef(t *test.T, saddr string) Calcer {
	client, e := rpc.DialHTTP("tcp", saddr)
	dieOnError(t,e)
	return NewRemoteCalcer(client)
}

func TestRemotize(t *test.T) {
	serveraddr:=fireUpCalcerServer(t)
	rcalc:=getRemoteRef(t,serveraddr)
	fmt.Println("-.5+2.7=",rcalc.Add(-.5, 2.7))
}



