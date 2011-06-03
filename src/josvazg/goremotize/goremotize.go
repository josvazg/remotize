package main

import (
	"flag"
	"josvazg/remotize"
)

func main() {
	flag.Parse()
	remotize.Autoremotize(".", flag.Args())
}

