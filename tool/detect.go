// Copyright 2011 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't have to rewrite an interface by
// hand in order to use it remotely. 
package tool

import (
	"bytes"
	"fmt"
	"github.com/josvazg/remotize"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

const (
	redefinedMarker = "\n// Redefined\n"
)

// Decl holds a type or interface source code declaration
type Decl struct {
	detected    *Detected
	isInterface bool
	Src         *bytes.Buffer
	imports     map[string]string
}

// remotizable detected info
//
// Detected will search for either:
//   * Calls refering to:
//     - Types OR Interfaces defined in some other package
//     - Types OR Interfaces declared in source code in current package
//   * Interface OR Type source declarations commented with "the (remotize) mark"
//
type Detected struct {
	packname   string
	aliases    map[string]string
	methods    map[string][]*ast.FuncDecl
	interfaces map[string]*ast.InterfaceType
	RDecls     map[string]*Decl
	RTypes     []string
}

/*
	Autoremotize will remotize all interfaces or types with methods that either:
	  - Are defined with a comment including '(remotize)' at the end
	  - Are used within certains calls like:
	    remotize.NewRemote(), remotize.NewService(), remotize.Please(), 
	    NewRemoteXXX() or NewXXXService()
*/
func Autoremotize(files ...string) (int, os.Error) {
	done := 0
	d, e := Detect(files...)
	if e != nil {
		return 0, e
	}
	items := len(d.RDecls) + len(d.RTypes)
	if items == 0 {
		fmt.Println("No 'remotizables' found")
		return done, nil
	}
	fmt.Printf("Found %v interfaces/types to remotize\n", items)
	e = buildRemotizer(d)
	if e != nil {
		return 0, e
	}
	return done, nil
}

// Detect will process go source files to detect interfaces or type 
// to be remotized
func Detect(files ...string) (*Detected, os.Error) {
	d := &Detected{}
	d.aliases = make(map[string]string)
	d.methods = make(map[string][]*ast.FuncDecl)
	d.interfaces = make(map[string]*ast.InterfaceType)
	d.RDecls = make(map[string]*Decl)
	d.RTypes = make([]string, 0)
	for _, f := range files {
		//fmt.Println("Parsing ", f, "?") 
		file, e := parser.ParseFile(token.NewFileSet(), f, nil,
			parser.ParseComments)
		if e != nil {
			fmt.Println(e)
			return nil, e
		}
		if d.packname == "" {
			d.packname = file.Name.Name
		} else if d.packname != file.Name.Name {
			panic("One package at a time! (can't remotize files from " +
				d.packname + " and " + file.Name.Name + " at the same time)")
		}
		//fmt.Println("Parsing ", f, "...")
		ast.Walk(d, file)
		//ast.Print(token.NewFileSet(), file)
	}
	postProcess(d)
	return d, nil
}

// Visit parses the whole source code
func (d *Detected) Visit(n ast.Node) (w ast.Visitor) {
	switch dcl := n.(type) {
	case *ast.ImportSpec:
		d.parseImports(dcl)
	case *ast.GenDecl:
		d.parseComment(dcl)
	case *ast.CallExpr:
		d.parseCalls(dcl)
	case *ast.FuncDecl:
		d.recordMethods(dcl)
	case *ast.TypeSpec:
		d.recordInterfaces(dcl)
	}
	return d
}

// parseImports will process imports for detection on each file's source code
func (d *Detected) parseImports(ispec *ast.ImportSpec) {
	path := strings.Trim(ispec.Path.Value, "\"")
	name := path2pack(path)
	current := path
	if ispec.Name != nil {
		current = ispec.Name.Name
	}
	previous, gotit := d.aliases[name]
	if gotit && previous != current {
		panic("Keep import aliases consistent within the package! Expected import " +
			name + " \"" + previous + "\" but got import " + name + "\"" + current + "\"!")
	} else if !gotit {
		d.aliases[name] = current
	}
}

// parseComment will search for interfaces or types in the source code preceeded 
// by a comment ending with '(remotize)' and will mark them for remotization
func (d *Detected) parseComment(decl *ast.GenDecl) {
	if decl.Doc == nil || decl.Specs == nil || len(decl.Specs) == 0 {
		return
	}
	tspec, ok := decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return
	}
	name := solveName(tspec.Name)
	i := len(decl.Doc.List) - 1
	for ; i >= 0 && empty(decl.Doc.List[i].Text); i-- {
	}
	if i >= 0 {
		cmt := decl.Doc.List[i]
		c := string(cmt.Text)
		if strings.Contains(strings.ToLower(c), "(remotize)") {
			if it, ok := tspec.Type.(*ast.InterfaceType); ok {
				d.interfaces[name] = it
				d.markType(name)
			} else {
				d.markType(name)
			}
		}
	}
}

// parseCalls will detect invocations of remotize calls like remotize.Please,
// remotize.NewRemote, remotize.NewServiceWith or NewRemoteXXX / NewXXXService
func (d *Detected) parseCalls(call *ast.CallExpr) {
	if call.Fun == nil {
		return
	}
	name := solveName(call.Fun)
	var called string
	if name == "" {
		return
	} else if name == "remotize.Please" {
		called = solveName(call.Args[0])
	} else if name == "remotize.NewRemote" || name == "remotize.NewServiceWith" {
		called = solveName(call.Args[1])
	} else if startsWith(name, "remotize.NewRemote") {
		called = name[len("remotize.NewRemote"):]
	} else if startsWith(name, "New") && endsWith(name, "Service") {
		called = name[len("New") : len(name)-len("Service")]
	} else {
		return
	}
	if called != "" {
		if startsWith(called, "*") {
			called = called[1:]
		}
		d.mark(called)
	}
}

// recordInterfaces will collect interface definition just in case there are 
// confirmed by some call as 'remotizable' 
func (d *Detected) recordInterfaces(tspec *ast.TypeSpec) {
	if it, ok := tspec.Type.(*ast.InterfaceType); ok {
		name := solveName(tspec.Name)
		if _, ok := d.interfaces[name]; !ok {
			d.interfaces[name] = it
		}
	}
}

// recordMethods will search for method Function Declarations in the source code
func (d *Detected) recordMethods(fdecl *ast.FuncDecl) {
	if fdecl.Recv == nil || fdecl.Name == nil ||
		!isExported(solveName(fdecl.Name)) {
		return
	}
	recv := solveName(fdecl.Recv.List[0])
	ml := d.methods[recv]
	if ml == nil {
		ml = make([]*ast.FuncDecl, 0)
	}
	ml = append(ml, fdecl)
	d.methods[recv] = ml
}

// markType marks an incomplete type by name. If it contains a package name
// is already defined on a another package and the name is enough. But if
// its from this package the methods must be discovered from source code
func (d *Detected) mark(name string) {
	if strings.Contains(name, ".") {
		d.RTypes = append(d.RTypes, name)
	} else {
		d.markType(name)
	}
}

// markType marks an type or interface given its name ready to hold methods
func (d *Detected) markType(name string) {
	d.RDecls[name] = &Decl{d, false, nil, nil}
	d.RDecls["*"+name] = &Decl{d, false, nil, nil}
}

// postProcess completes candidate types with their methods (retrieved by 
// parseMethods) and pass them and the interfaces found as sources within Detected
func postProcess(d *Detected) {
	for name, dcl := range d.RDecls { // complete types with methods
		if d.interfaces[name] == nil && d.methods[name] != nil { // is a type declaration
			methods := bytes.NewBufferString("")
			for _, fdecl := range d.methods[name] { // for each type's method...
				ast.Walk(dcl, fdecl) // -> call dcl.Visit
				method := solveName(fdecl.Name)
				tmpbuf := bytes.NewBufferString("")
				printer.Fprint(tmpbuf, token.NewFileSet(), fdecl.Type)
				signature := tmpbuf.String()[4:]
				fmt.Fprintf(methods, "\n\t%s%s", method, signature)
			}
			dcl.Src = bytes.NewBufferString("")
			//fmt.Fprintf(dcl.Src, "%s", header(d.packname, dcl.imports))
			fmt.Fprintf(dcl.Src, "\ntype %s interface {", ifacename(name))
			fmt.Fprintf(dcl.Src, "%s", methods)
			fmt.Fprintf(dcl.Src, "\n}\n")
			dcl.isInterface = false
		} else if it := d.interfaces[name]; it != nil { // is a interface declaration
			ast.Walk(dcl, it) // -> call dcl.Visit
			dcl.Src = bytes.NewBufferString("")
			//fmt.Fprintf(dcl.Src, "%s", header(d.packname, dcl.imports))
			fmt.Fprintf(dcl.Src, "%s", redefinedMarker)
			fmt.Fprintf(dcl.Src, "type %s ", name)
			printer.Fprint(dcl.Src, token.NewFileSet(), it)
			fmt.Fprintf(dcl.Src, "\n")
			dcl.isInterface = true
		}
		if dcl.Src == nil {
			d.RDecls[name] = nil, false
		}
	}
}

// Visit parses a candidate interface source code
func (dcl *Decl) Visit(n ast.Node) (w ast.Visitor) {
	switch t := n.(type) {
	case ast.Expr:
		name := solveName(t)
		if strings.Contains(name, ".") {
			if dcl.imports == nil {
				dcl.imports = make(map[string]string, 0)
			}
			packname := strings.Split(name, ".")[0]
			dcl.imports[packname] = dcl.detected.aliases[packname]
		}
	case *ast.BlockStmt:
		return nil
	}
	return dcl
}

// solveName is given an ast node and tries to solve its name
func solveName(e interface{}) string {
	if e == nil {
		return ""
	}
	switch (e).(type) {
	case *ast.Field:
		return solveName((interface{})(e).(*ast.Field).Type)
	case *ast.Ident:
		return (interface{})(e).(*ast.Ident).Name
	case *ast.StarExpr:
		return "*" + solveName((interface{})(e).(*ast.StarExpr).X)
	case *ast.SelectorExpr:
		se := (interface{})(e).(*ast.SelectorExpr)
		return solveName(se.X) + "." + se.Sel.Name
	case *ast.CallExpr:
		call := e.(*ast.CallExpr)
		name := solveName(call.Fun)
		if name == "new" {
			return solveName(call.Args[0])
		}
		return name
	case *ast.UnaryExpr:
		ue := (e).(*ast.UnaryExpr)
		prefix := ""
		if ue.Op == token.AND {
			prefix = "*"
		}
		return prefix + solveName(ue.X)
	case *ast.BasicLit:
		bl := (e).(*ast.BasicLit)
		return bl.Value
	case *ast.CompositeLit:
		cl := (e).(*ast.CompositeLit)
		if cl.Type != nil {
			return solveName(cl.Type)
		} else if cl.Elts != nil && len(cl.Elts) > 0 {
			return solveName(cl.Elts[0])
		}
	}
	return ""
}

// startsWith returns true if str starts with substring s
func startsWith(str, s string) bool {
	return len(str) >= len(s) && str[:len(s)] == s
}

// endsWith returns true if str ends with substring s
func endsWith(str, s string) bool {
	return len(str) >= len(s) && str[len(str)-len(s):] == s
}

// ifaceName returns the correspondent interface name for a given type
func ifacename(name string) string {
	return strings.TrimLeft(name, " *") + remotize.Suffix(name)
}

// isExported returns true if the given FuncDecl is Exported (=Title-case)
func isExported(name string) bool {
	return name != "" && name[0:1] == strings.ToUpper(name[0:1])
}

// empty tells whether the string really empty or not
func empty(s string) bool {
	s = strings.Trim(s, " \t")
	return len(s) == 0 || s == "//" || s == "*/"
}

// returns the last part of the path as the package name or the full path if it's just a single name path
func path2pack(path string) string {
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		return parts[len(parts)-1]
	}
	return path
}

// gofmtSave saves the go source to a file properly formatted
func gofmtSave(filename, source string) os.Error {
	fset := token.NewFileSet()
	f, e := parser.ParseFile(fset, filename+".go", source, parser.ParseComments)
	if e != nil {
		fmt.Println(source)
		return e
	}
	fos, e := os.Create(filename + ".go")
	if e != nil {
		return e
	}
	pcfg := &printer.Config{printer.TabIndent, 2}
	pcfg.Fprint(fos, fset, f)
	fos.Close()
	_, e = os.Stat(filename + ".go")
	return e
}
