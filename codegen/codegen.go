package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

type GoFile struct {
	astFile		*ast.File
	filename	string
	header		string
}

func NewGoFile(name, pack string) *GoFile {
	code:="package "+pack+"\n\n"
	n,e:=parse(nil,code)
	if(e!=nil) {
		panic(e)
	}
	gofile:=&GoFile{n,name,code}
	return gofile
}

func (gf *GoFile) AddImport(name string) os.Error {
	return gf.parse("import \""+name+"\"\n")
}

func (gf *GoFile) AddAliasedImport(name, alias string) os.Error {
	return gf.parse("import "+alias+" \""+name+"\"\n")
}

func (gf *GoFile) AddFunc(funcDecl string) os.Error {
	return gf.parse(funcDecl)
}

func parse(f *GoFile, code string) (*ast.File,os.Error) {
	if(f!=nil) {
		code=f.header+code;
	}
	fset:=token.NewFileSet()
	return parser.ParseFile(fset,"",code,0)
}

func (gf *GoFile) parse(code string) os.Error {
	n,e:=parse(gf,code)
	if(e!=nil) {
		return e
	}
	gf.astFile.Decls=append(gf.astFile.Decls,n.Decls...)
	return nil
}

