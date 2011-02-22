package codegen

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"testing"
)

func TestCodegen(t *testing.T) {
	fmt.Println("Generating go code sample")
	gf:=NewGoFile("gogen.go","gogen")
	e:=gf.AddImport("fmt")
	if(e!=nil) {
		fmt.Println(e)
	}
	e=gf.AddAliasedImport("testing","test")
	if(e!=nil) {
		fmt.Println(e)
	}
	e=gf.AddFunc(`func someFunc(a, b int) int {
		return a+b
	}`)	
	if(e!=nil) {
		fmt.Println(e)
	}
	//fmt.Println("Replace importSpec fmt",gf.ReplaceImport("fmt","f"))

	fmt.Println(gf)
	ast.Print(gf.astFile)
	fmt.Println("Source code:")
	printer.Fprint(os.Stdout,token.NewFileSet(),gf.astFile)
}
