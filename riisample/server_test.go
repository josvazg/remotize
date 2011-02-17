package riisample

import (
	"fmt"
	"testing"
	"os"
)

func runServer() {
	gobSync(os.Stdin,os.Stdout)
	cs:=newCalcServer(os.Stdin,os.Stdout,&simplecalc{0})
	cs.Serve()
	fmt.Fprintln(os.Stderr,"Server done")	
}

func TestServer(t *testing.T) {
	run:=os.Getenv("RUN_RIISAMPLE_SERVER")
	if(run=="RUN") {
		runServer()
	}
}
