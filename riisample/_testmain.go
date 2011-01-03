package main

import "riisample"
import "testing"
import __regexp__ "regexp"

var tests = []testing.InternalTest{
	{"riisample.TestServer", riisample.TestServer},
}
var benchmarks = []testing.InternalBenchmark{}

func main() {
	testing.Main(__regexp__.MatchString, tests)
	testing.RunBenchmarks(__regexp__.MatchString, benchmarks)
}
