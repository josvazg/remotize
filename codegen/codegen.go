package codegen

import (
	"fmt"
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
	return gf.replaceImport(name,"")
	//return gf.parse("import \""+name+"\"\n")
}

func (gf *GoFile) AddAliasedImport(name, alias string) os.Error {
	return gf.replaceImport(name,alias)
	//return gf.parse("import "+alias+" \""+name+"\"\n")
}

func (gf *GoFile) replaceImport(name, alias string) os.Error {
	target:="\""+name+"\""
	newone,e:=gf.importSpec(name,alias)
	if(e!=nil) {
		return e
	}
	var imports *ast.GenDecl
	imports=nil
	for _,decl := range gf.astFile.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for pos, specs := range d.Specs {
				switch s := specs.(type) {
				case *ast.ImportSpec:
					if(imports==nil) {
						imports=d
					}
					val:=string(s.Path.Value)
					if(val==target) {
						d.Specs[pos]=newone
						return	nil
					}
				}
			}						
		}
	}	
	if(imports==nil) {
		return gf.parse(importCode(name,alias))
	}
	imports.Specs=append(imports.Specs,newone)
	imports.Lparen=1 // activate parenthesis for the import list
	return nil	
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

func DebugCode(code string) {
	fmt.Println(code)
	fmt.Println("AST:")
	fset:=token.NewFileSet()
	n,e:=parser.ParseFile(fset,"",code,0)
	if(e!=nil) {
		fmt.Println(e)
	} else {
		ast.Print(n)
	}
}

func (gf *GoFile) importSpec(name,alias string) (*ast.ImportSpec, os.Error) {
	n,e:=parse(gf,importCode(name,alias))
	if(e!=nil) {
		return nil,e
	}
	return ((n.Decls[0]).(*ast.GenDecl).Specs[0]).(*ast.ImportSpec),nil	
}

func importCode(name, alias string) string {
	if(alias=="") {
		return "import \""+name+"\"\n"
	}
	return "import "+alias+" \""+name+"\"\n"
}
