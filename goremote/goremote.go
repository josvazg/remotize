// The goremote command tries to remotize types from a bunch of files on the same package.
//
// Usage: 
//   goremote <list of go files, *.go...>
package main

import (
	"flag"
	"fmt"
	"github.com/josvazg/remotize/tool"
	"os"
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

// autoremotize will remotize all interfaces or types detected be remotizable within given files
func autoremotize(files ...string) (int, os.Error) {
	done := 0
	d, e := tool.Detect(files...)
	if e != nil {
		return 0, e
	}
	if d == nil || len(d) == 0 {
		fmt.Println("No 'remotizables' found")
		return done, nil
	}
	fmt.Printf("Found %v interfaces/types to remotize\n", len(d))
	e = tool.BuildRemotizer(d)
	if e != nil {
		return 0, e
	}
	return done, nil
}

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	files:=filterRemotized(flag.Args())
	if len(flag.Args()) > 0 {
		fmt.Println("remotize/goremote is scanning", files , "...")
		autoremotize(files...)
		fmt.Println("remotize/goremote tool ends")
	} else {
		fmt.Println("No source files provided to remotize/goremote!")
		fmt.Println("Usage: goremote <list of go files, *.go...>")
	}
}

