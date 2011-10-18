package main

import (
	"flag"
	"fmt"
	"github.com/josvazg/remotize/tool"
)

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Println("remotize/goremote is scanning", flag.Args(), "...")
		tool.Autoremotize(flag.Args()...)
	} else {
		fmt.Println("No source files provided to remotize/goremote!")
		fmt.Println("Usage: goremote <list of go files, *.go...>")
	}
}

