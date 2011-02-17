package riisample

import (
	"fmt"
	"testing"
	"os"
	"rii"
)

func runServer() {
	cs:=newCalcServer(&simplecalc{0})
	cs.ServePipe(rii.IO(os.Stdin,os.Stdout))
	fmt.Fprintln(os.Stderr,"Server done")	
}

func TestServer(t *testing.T) {
	run:=os.Getenv("RUN_RIISAMPLE_SERVER")
	if(run=="RUN") {
		runServer()
	}
}
