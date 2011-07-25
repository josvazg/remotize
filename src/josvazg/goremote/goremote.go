package main

import (
	"flag"
	"remotize/tool"
)

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	tool.Autoremotize(flag.Args()...)
}

