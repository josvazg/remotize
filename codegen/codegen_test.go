package codegen

import (
	"fmt"
	"testing"
)

func TestCodegen(t *testing.T) {
	fmt.Println("Generating go code sample")
	gf:=NewGoFile("gogen.go","gogen")
	gf.AddImport("fmt")
	gf.AddAliasedImport("testing","test")
	fmt.Println(gf)
}
