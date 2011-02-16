package main

import "codegen"
import "testing"
import __regexp__ "regexp"

var tests = []testing.InternalTest{
	{"codegen.TestCodegen", codegen.TestCodegen},
}
var benchmarks = []testing.InternalBenchmark{}

func main() {
	testing.Main(__regexp__.MatchString, tests)
	testing.RunBenchmarks(__regexp__.MatchString, benchmarks)
}
