package sample

import (
	"fmt"
	"github.com/josvazg/remotize"
	"http"
	"net"
	test "testing"
)

func dieOnError(e os.Error) {
	if e != nil {
		t.Fatal("listen error:", e)
	}
}

func fireUpCalcerServer(t *test.T) string {
	server := rpc.NewServer()
	// You can access the remotized code directly, it should be created by now...
	calcer:=remotize.NewCalcerService(server,new(Calc))
	if calcer == nil {
		t.Fatal("Calcer server could NOT be created!")
	}
	addr=":1234"
	l, e := net.Listen("tcp", addr)
	dieOnError(e)
	go http.Serve(l, nil)
	return addr
}

func getRemoteRef(saddr string) Calcer {
	client, e := rpc.DialHTTP("tcp", saddr)
	dieOnError(e)
	return remotize.NewRemoteCalcer(client)
}

func TestRemotize(t *test.T) {
	serveraddr:=fireUpCalcerServer()
	rcalc:=getRemoteRef(serveraddr)
	fmt.Println("-.5+2.7=",rcalc.Add(-.5, 2.7))
}



