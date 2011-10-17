package main

import (
	"flag"
	"github.com/josvazg/remotize/tool"
)

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	tool.Autoremotize(flag.Args()...)
}

