package rii

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/printer"
	"os"
	"strings"
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

func addLineComment(cg *ast.CommentGroup, comment string) *ast.CommentGroup {
	lines:=strings.Split(comment, "\n", -1)
	if(cg==nil) {
		cg=&ast.CommentGroup{nil}
	}
	for _,line := range lines {
		fmt.Println("line:"+line)
		cg.List=append(cg.List,&ast.Comment{token.Pos(1),
			([]byte)("//"+line+"\n")})
	}
	return cg
}

func (gf *GoFile) AddLineComment(comment string) os.Error {
	addLineComment(gf.astFile.Doc,comment)
	fmt.Println("n.Doc:",gf)
	ast.Print(gf)
	printer.Fprint(os.Stdout, token.NewFileSet(), gf.astFile)
	return nil
}

func (gf *GoFile) AddImport(name string) os.Error {
	return gf.replaceImport(name,"")
}

func (gf *GoFile) AddAliasedImport(name, alias string) os.Error {
	return gf.replaceImport(name,alias)
}

func (gf *GoFile) DeclType(name, typedecl string) os.Error {
	// TODO check if already defined
	return gf.parse("type "+name+" "+typedecl)
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
	return parser.ParseFile(token.NewFileSet(),"",code,0)
}

func (gf *GoFile) parse(code string) os.Error {
	n,e:=parse(gf,code)
	if(e!=nil) {
		return e
	}
	gf.astFile.Decls=append(gf.astFile.Decls,n.Decls...)
	return nil
}

func (gf *GoFile) Debug() {
	ast.Print(gf.astFile)
	printer.Fprint(os.Stdout, token.NewFileSet(), gf.astFile)
}

func Debug(code string) {
	f,e:=parser.ParseFile(token.NewFileSet(),"",code,parser.ParseComments)
	if(e!=nil) {
		fmt.Print(e)
		return
	}
	f.Comments=nil
	ast.Print(f)
	prc:=&printer.Config{2,4}
	prc.Fprint(os.Stdout, token.NewFileSet(), f)
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
