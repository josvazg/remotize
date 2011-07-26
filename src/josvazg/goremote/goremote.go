package main

import (
	"flag"
	"josvazg/remotize/tool"
)

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	tool.Autoremotize(flag.Args()...)
}

