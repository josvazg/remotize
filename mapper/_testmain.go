package main

import "container/mapper"
import "testing"
import __regexp__ "regexp"

var tests = []testing.Test{
	{"mapper.TestMap", mapper.TestMap},
	{"mapper.TestThreadSafeMap", mapper.TestThreadSafeMap},
}
var benchmarks = []testing.InternalBenchmark{}

func main() {
	testing.Main(__regexp__.MatchString, tests)
	testing.RunBenchmarks(__regexp__.MatchString, benchmarks)
}
