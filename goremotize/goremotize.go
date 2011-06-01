package main

import (
	"flag"
	"remotize"
)

func main() {
	flag.Parse()
	remotize.Autoremotize(".", flag.Args())
}

