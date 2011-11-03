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
	specSeparator   = ":\n"
)

// Candidate holds a type or interface candidate to be remotized (or not)
/*type candidate struct {
	state int
	value interface{}
	packs []string
}*/

// remotizable detected info
type Detected struct {
	currpack   string
	aliases    map[string]string
	methods    map[string][]*ast.FuncDecl
	interfaces map[string]*ast.InterfaceType
	RInterfaces      map[string]*ast.InterfaceType
	RDeclarations map[]string]*declaration
	RTypes	[]string
	/*candidates map[string]*candidate
	sources    map[string]string
	types      []string*/
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
	d, e := detect(files...)
	if e != nil {
		fmt.Println("e:", e)
		return 0, e
	}
	fmt.Println("Detected:", d)
	items := len(d.sources) + len(d.types)
	if items == 0 {
		fmt.Println("No 'remotizables' found")
		return done, nil
	}
	fmt.Printf("Found %v interfaces/types to remotize:\n", items)
	e = buildRemotizer(d)
	if e != nil {
		fmt.Println("Error:", e)
	}
	return done, nil
}

// detect will process go source files to detect interfaces or type 
// to be remotized
func detect(files ...string) (*Detected, os.Error) {
	d := &Detected{}
	d.aliases = make(map[string]string)
	d.methods = make(map[string][]*ast.FuncDecl)
	d.interfaces = make(map[string]*ast.InterfaceType)
	d.RInterfaces = make(map[string]*ast.InterfaceType)
	d.RDeclarations = make(map[string]*declaration)
	d.RTypes = make([]string],0)
	for _, f := range files {
		//fmt.Println("Parsing ", f, "?") 
		file, e := parser.ParseFile(token.NewFileSet(), f, nil,
			parser.ParseComments)
		if e != nil {
			fmt.Println(e)
			return nil,e
		}
		if d.currpack == "" {
			d.currpack = file.Name.Name
		} else if d.currpack != file.Name.Name {
			panic("One package at a time! (can't remotize files from " +
				d.currpack + " and " + file.Name.Name + " at the same time)")
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
	case *ast.GenDecl:
		d.parseComment(dcl)
	case *ast.FuncDecl:
		d.parseMethods(dcl)
	case *ast.CallExpr:
		d.parseCalls(dcl)
	case *ast.TypeSpec:
		d.parseTypes(dcl)
	case *ast.ImportSpec:
		d.parseImports(dcl)
	}
	return d
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
				fmt.Println("Interface from comments:", name)
				d.RInterfaces[name] = it
			} else {
				fmt.Println("Type from comments: (*)", name)
				d.markType(name)
			}
		}
	}
}

// parseTypes will collect interface definition just in case there are 
// confirmed by some call as 'remotizable' 
func (d *Detected) parseTypes(tspec *ast.TypeSpec) {
	if it, ok := tspec.Type.(*ast.InterfaceType); ok {
		name := solveName(tspec.Name)
		if _, ok := d.RInterfaces[name]; !ok {
			if _, ok := d.interfaces[name]; !ok {
				fmt.Println("Purposed interface:", name)
				d.interfaces[name] = it
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
		if it, ok := d.interfaces[called]; ok {
			fmt.Println("Interface from calls:", called, "value", it)
			d.RInterfaces[called]=it
		} else {
			fmt.Println("Type or Interface Candidate from calls:", called)
			d.mark(called)
		}
	}
}

// parseMethods will search for method Function Declarations in the source code
func (d *Detected) parseMethods(fdecl *ast.FuncDecl) {
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
	fmt.Println("Recorded method for", recv, ":", *fdecl)
}

// parseImports will process imports for detection on each file's source code
func (d *Detected) parseImports(ispec *ast.ImportSpec) {
	path := strings.Trim(ispec.Path.Value, "\"")
	name := path
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		name = parts[len(parts)-1]
	}
	if ispec.Name != nil {
		d.aliases[name] = ispec.Name.Name
	} else if name != path {
		d.aliases[name] = path
	}
}

// postProcess completes candidate types with their methods (retrieved by 
// parseMethods) and pass them and the interfaces found as sources within Detected
func postProcess(d *Detected) {
	for name, dcl := range d.RDeclarations { // complete types with methods
		if d.methods[name] != nil {
			for _, fdecl := range d.methods[name] {
				method := solveName(fdecl.Name)
				ast.Walk(dcl, fdecl)
				tmpbuf := bytes.NewBufferString("")
				printer.Fprint(tmpbuf, token.NewFileSet(), fdecl.Type)
				signature := tmpbuf.String()[4:]
				fmt.Fprintf(dcl.src, "\n\t%s%s", method, signature)
			}
			fmt.Fprintf(src, "\n}\n")
			tmpbuf := bytes.NewBufferString("")
			fmt.Fprintf(tmpbuf, "%s%s",header(d,name),dcl.src.String())
			dcl.src=tmpbuf
		}
	}
	for name, it := range d.interfaces {
		if it!=nil {
			ast.Walk(dcl, it)
			src := bytes.NewBufferString("type " + name + " ")
			printer.Fprint(src, token.NewFileSet(), it)
			fmt.Fprintf(src, "\n")
			d.sources[name] = ifacename(name) + specSeparator + header(d, name) + 
					redefinedMarker + src.String()
		}
	}
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

// header generates the package and imports header for a candidate
/*func header(d *Detected, name string) string {
	c := d.candidates[name]
	tmpbuf := bytes.NewBufferString(ifacename(name) + specSeparator +
		"package " + d.currpack + "\n\n")
	c.packs=append(c.packs,"rpc")
	if c.packs != nil && len(c.packs) > 0 {
		fmt.Fprintf(tmpbuf, "import (\n")
		for _, pack := range c.packs {
			if a := d.aliases[pack]; a != "" {
				fmt.Fprintf(tmpbuf, "\t%v \"%v\"\n", pack, a)
			} else {
				fmt.Fprintf(tmpbuf, "\t\"%v\"\n", pack)
			}
		}
		fmt.Fprintf(tmpbuf, ")\n\n")
	}
	return tmpbuf.String()
}*/

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
	d.RDeclarations[name] = newDeclaration(nil,"type " + ifacename(name) + " interface {")
	d.RDeclarations["*"+name] = newDeclaration(nil,"type " + ifacename(name) + " interface {")
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

// Visit parses a candidate interface source code
func (d *declaration) Visit(n ast.Node) (w ast.Visitor) {
	switch t := n.(type) {
	case ast.Expr:
		name := solveName(t)
		if strings.Contains(name, ".") {
			if d.imports == nil {
				d.imports = make(map[string]string, 0)
			}
			parts := strings.Split(name, ".")
			for _, p := range d.imports {
				if p == parts[0] {
					return
				}
			}
			d.imports[parts[0]] = parts[0]
		}
	case *ast.BlockStmt:
		return nil
	}
	return c
}

