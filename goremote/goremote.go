package main

import (
	"flag"
	"fmt"
	"github.com/josvazg/remotize/tool"
)

// filterRemotized will take out the remotized*.go occurrences if any
func filterRemotized(names []string) []string {
	result:=names
	changed:=false
	for n,name:=range names {		
		if name=="remotized*.go" && !changed {
			changed=true
			if n>0 {
				result=names[:n]
			} else {
				result=nil
			}			
		} else if name!="remotized*.go" && changed {			
			result=append(result,name)
		}		
	}
	return result
}

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Println("remotize/goremote is scanning", filterRemotized(flag.Args()), "...")
		tool.Autoremotize(flag.Args()...)
		fmt.Println("remotize/goremote tool ends")
	} else {
		fmt.Println("No source files provided to remotize/goremote!")
		fmt.Println("Usage: goremote <list of go files, *.go...>")
	}
}

