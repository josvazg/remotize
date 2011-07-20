package remotize

import (
	"bytes"
	"exec"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sort"
	"strings"
)

// Candidate States
const (
	ProposedInterface = iota
	Type
	Interface
)

// remotizer code head and tail & marker
const (
	remotizerHead   = `// Autogenerated Remotizer [DO NOT EDIT!]
package main
`
	remotizerTail   = `func main() {
	for _,r := range toremotize {
		if e:=remotize.Remotize0(r); e!=nil {
			panic(e)
		}
	}
}`
	redefinedMarker = "\n// Redefined\n"
	specSeparator   = ":\n"
)

// Candidate holds a type or interface candidate to be remotized (or not)
type candidate struct {
	state int
	value interface{}
	packs []string
}

// remotize spec
type rinfo struct {
	currpack   string
	aliases    map[string]string
	methods    map[string][]*ast.FuncDecl
	candidates map[string]*candidate
	sources    map[string]string
	types      []string
}

// suppress will remove the ocurrences of sups strings from s 
// and return the result
func suppress(s string, sups ...string) string {
	for _, sup := range sups {
		s = strings.Replace(s, sup, "", -1)
	}
	return s
}

// empty tells whether the string really empty or not
func empty(s string) bool {
	s = strings.Trim(s, " \t")
	return len(s) == 0 || s == "//" || s == "*/"
}

// fixPack will fix the package name to be present and without alias
func fixPack(r *rinfo, name string) string {
	if !strings.Contains(name, ".") {
		return r.currpack + "." + name
	}
	parts := strings.Split(name, ".")
	alias := r.aliases[parts[0]]
	if alias == "" {
		return name
	}
	return alias + "." + parts[1]
}

// ifaceName returns the correspondent interface name for a given type
func ifacename(name string) string {
	return strings.TrimLeft(name, " *") + suffix(name)
}

// markType marks an incomplete type by name. If it contains a package name
// is already defined on a another package and the name is enough. But if
// its from this package the methods must be discovered from source code
func mark(r *rinfo, name string) {
	if strings.Contains(name, ".") {
		r.types = append(r.types, name)
		return
	}
	markType(r, name)
}

// markType marks an type or interface given its name ready to hold methods
func markType(r *rinfo, name string) {
	typesrc := bytes.NewBufferString("type " + ifacename(name) + " interface {")
	r.candidates[name] = &candidate{Type, typesrc, nil}
	//fmt.Println("Incomplete type:", name)
}

// parseComment will search for interfaces or types in the source code preceeded 
// by a comment ending with '(remotize)' and will mark them for remotization
func parseComment(r *rinfo, decl *ast.GenDecl) {
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
				r.candidates[name] = &candidate{Interface, it, nil}
			} else {
				fmt.Println("Type from comments: (*)", name)
				markType(r, name)
				markType(r, "*"+name)
			}
		}
	}
}

// parseTypes will collect interface definition just in case there are 
// confirmed by some call as 'remotizable' 
func parseTypes(r *rinfo, tspec *ast.TypeSpec) {
	if it, ok := tspec.Type.(*ast.InterfaceType); ok {
		name := solveName(tspec.Name)
		if _, ok := r.candidates[name]; !ok {
			r.candidates[name] = &candidate{ProposedInterface, it, nil}
			fmt.Println("Proposed interface:", name)
		}
	}
}

// parseCalls will detect invocations of remotize calls like remotize.Please,
// remotize.NewRemote, remotize.NewServiceWith or NewRemoteXXX / NewXXXService
func parseCalls(r *rinfo, call *ast.CallExpr) {
	if call.Fun == nil {
		return
	}
	name := solveName(call.Fun)
	var called string
	if name == "Please" {
		called = solveName(call.Args[0])
	} else if name == "NewRemote" || name == "NewServiceWith" {
		called = solveName(call.Args[1])
	} else if startsWith(name, "NewRemote") {
		called = name[len("NewRemote"):]
	} else if startsWith(name, "New") && endsWith(name, "Service") {
		called = name[len("New") : len(name)-len("Service")]
	} else {
		return
	}
	if called != "" {
		if can, ok := r.candidates[called]; ok {
			if can.state == ProposedInterface {
				can.state = Interface
				fmt.Println("Interface from calls:", called, "value", can.value)
			}
		} else {
			fmt.Println("Type or Interface Candidate from calls:", called)
			mark(r, called)
		}
	}
}

// parseMethods will search for method Function Declarations in the source code
func parseMethods(r *rinfo, fdecl *ast.FuncDecl) {
	if fdecl.Recv == nil || fdecl.Name == nil ||
		!isExported(solveName(fdecl.Name)) {
		return
	}
	recv := solveName(fdecl.Recv.List[0])
	ml := r.methods[recv]
	if ml == nil {
		ml = make([]*ast.FuncDecl, 0)
	}
	ml = append(ml, fdecl)
	r.methods[recv] = ml
	fmt.Println("Recorded method for", recv, ":", *fdecl)
}

// parseImports will process imports for detection on each file's source code
func parseImports(r *rinfo, ispec *ast.ImportSpec) {
	path := strings.Trim(ispec.Path.Value, "\"")
	name := path
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		name = parts[len(parts)-1]
	}
	if ispec.Name != nil {
		r.aliases[name] = ispec.Name.Name
		fmt.Println("import aliases", r.aliases)
	} else if name != path {
		r.aliases[name] = path
		fmt.Println("* import aliases", r.aliases)
	}
}

// Visit parses a candidate interface source code
func (c *candidate) Visit(n ast.Node) (w ast.Visitor) {
	switch d := n.(type) {
	case ast.Expr:
		name := solveName(d)
		if strings.Contains(name, ".") {
			if c.packs == nil {
				c.packs = make([]string, 0)
			}
			parts := strings.Split(name, ".")
			for _, p := range c.packs {
				if p == parts[0] {
					return
				}
			}
			c.packs = append(c.packs, parts[0])
		}
	case *ast.BlockStmt:
		return nil
	}
	return c
}

// Visit parses the whole source code
func (r *rinfo) Visit(n ast.Node) (w ast.Visitor) {
	switch d := n.(type) {
	case *ast.GenDecl:
		parseComment(r, d)
	case *ast.FuncDecl:
		parseMethods(r, d)
	case *ast.CallExpr:
		parseCalls(r, d)
	case *ast.TypeSpec:
		parseTypes(r, d)
	case *ast.ImportSpec:
		parseImports(r, d)
	}
	return r
}

// header generates the package and imports header for a candidate
func header(r *rinfo, name string) string {
	c := r.candidates[name]
	tmpbuf := bytes.NewBufferString(ifacename(name) + specSeparator +
		"package " + r.currpack + "\n\n")
	if c.packs != nil && len(c.packs) > 0 {
		fmt.Fprintf(tmpbuf, "import (\n")
		for _, pack := range c.packs {
			if a := r.aliases[pack]; a != "" {
				fmt.Fprintf(tmpbuf, "\t%v \"%v\"\n", pack, a)
			} else {
				fmt.Fprintf(tmpbuf, "\t\"%v\"\n", pack)
			}
		}
		fmt.Fprintf(tmpbuf, ")\n\n")
	}
	return tmpbuf.String()
}

// postProcess completes candidate types with their methods (retrieved by 
// parseMethods) and pass them and the interfaces found as sources within rinfo
func postProcess(r *rinfo) {
	for name, can := range r.candidates { // complete types with methods
		if can.state == Type && can.value != nil {
			src := can.value.(*bytes.Buffer)
			if r.methods[name] != nil {
				for _, fdecl := range r.methods[name] {
					method := solveName(fdecl.Name)
					ast.Walk(can, fdecl)
					tmpbuf := bytes.NewBufferString("")
					printer.Fprint(tmpbuf, token.NewFileSet(), fdecl.Type)
					signature := tmpbuf.String()[4:]
					fmt.Fprintf(src, "\n\t%s%s", method, signature)
				}
				fmt.Fprintf(src, "\n}\n")
				r.sources[name] = header(r, name) + src.String()
			}
		}
	}
	for name, can := range r.candidates {
		if can.state == Interface || can.state == Type {
			if it, ok := can.value.(*ast.InterfaceType); ok {
				ast.Walk(can, it)
				src := bytes.NewBufferString("type " + name + " ")
				printer.Fprint(src, token.NewFileSet(), it)
				fmt.Fprintf(src, "\n")
				r.sources[name] = header(r, name) + redefinedMarker + src.String()
			}
		}
	}
}

// parseFiles will process go source files to detect interfaces or type 
// to be remotized
func parseFiles(r *rinfo, files ...string) os.Error {
	fmt.Println("About to parse ", files)
	for _, f := range files {
		fmt.Println("Parsing ", f, "?")
		file, e := parser.ParseFile(token.NewFileSet(), f, nil,
			parser.ParseComments)
		if e != nil {
			fmt.Println(e)
			return e
		}
		if r.currpack == "" {
			r.currpack = file.Name.Name
		} else if r.currpack != file.Name.Name {
			panic("One package at a time! (can't remotize files from " +
				r.currpack + " and " + file.Name.Name + " at the same time)")
		}
		fmt.Println("Parsing ", f, "...")
		ast.Walk(r, file)
		//ast.Print(token.NewFileSet(), file)
	}
	postProcess(r)
	return nil
}

// addImports adds imports to the remotizer source code from types
func addImports(r *rinfo, imports []string) []string {
loop:
	for _, typename := range r.types {
		parts := strings.SplitN(typename, ".", 2)
		packname := parts[0]
		for _, imp := range imports {
			if packname == imp {
				continue loop
			}
		}
		imports = append(imports, packname)
	}
	return imports
}

// generateRemotizerCode returns the remotizer source code for a given rinfo
func generateRemotizerCode(r *rinfo) string {
	src := bytes.NewBuffer(make([]byte, 0))
	fmt.Fprintf(src, remotizerHead)
	fmt.Fprintf(src, "import (\n")
	imports := []string{"remotize"}
	imports = addImports(r, imports)
	sort.Strings(imports)
	for _, s := range imports {
		if fullpath, ok := r.aliases[s]; ok && fullpath != s {
			fmt.Fprintf(src, "%s \"%s\"\n", s, fullpath)
		} else {
			fmt.Fprintf(src, "\"%s\"\n", s)
		}
	}
	fmt.Fprintf(src, ")\n\n")
	fmt.Fprintf(src, "var toremotize = []interface{}{\n")
	for _, s := range r.sources {
		fmt.Fprintf(src, "`%v`,", s)
	}
	for _, s := range r.types {
		fmt.Fprintf(src, "\nnew(%v),", s)
	}
	fmt.Fprintf(src, "\n}\n\n")
	fmt.Fprintf(src, remotizerTail)
	return src.String()
}

// writeAndFormatSource writes the go source to a file properly formatted
func writeAndFormatSource(filename, source string) os.Error {
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

// build generates a program to remotize the detected interfaces
func buildRemotizer(r *rinfo) os.Error {
	src := generateRemotizerCode(r)
	fmt.Println(src)
	filename := "_remotizer"
	if e := writeAndFormatSource(filename, src); e != nil {
		return e
	}
	if o, e := runCmd(gocompile(), "-I", "_test", filename+".go"); e != nil {
		fmt.Fprintf(os.Stderr, string(o)+"\n")
		return e
	}
	if o, e := runCmd(golink(), "-L", "_test", "-o", filename,
		filename+"."+goext()); e != nil {
		fmt.Fprintf(os.Stderr, string(o)+"\n")
		return e
	}
	if o, e := runCmd("./" + filename); e != nil {
		fmt.Fprintf(os.Stderr, string(o)+"\n")
		return e
	}
	return nil
}

/*
	Autoremotize will remotize all interfaces that either:
	- Are defined with a comment including '(remotize)' at the end
	- Are used within certains calls like: remotize.NewRemote(), 
remotize.NewService(), remotize.Please(), NewRemoteXXX() or NewXXXService()
*/
func Autoremotize(files ...string) (int, os.Error) {
	done := 0
	rs := &rinfo{}
	rs.aliases = make(map[string]string)
	rs.candidates = make(map[string]*candidate)
	rs.methods = make(map[string][]*ast.FuncDecl)
	rs.sources = make(map[string]string)
	rs.types = make([]string, 0)
	if e := parseFiles(rs, files...); e != nil {
		return 0, e
	}
	items := len(rs.sources) + len(rs.types)
	if items == 0 {
		fmt.Println("No 'remotizables' found")
		return done, nil
	}
	fmt.Printf("Found %v interfaces/types to remotize:\n", items)
	e := buildRemotizer(rs)
	if e != nil {
		fmt.Println("Error:", e)
	}
	return done, nil
}

// solveName is given an ast node and tries to solve its name
func solveName(e interface{}) string {
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

// runs a command
func runCmd(cmdargs ...string) ([]byte, os.Error) {
	fmt.Println(cmdargs)
	return exec.Command(cmdargs[0], cmdargs[1:]...).CombinedOutput()
}

// dictionary cache
var dict map[string]string

// go tool execution string
func goexec(tool string) string {
	if dict == nil {
		dict = make(map[string]string)
		dict["386"] = "8"
		dict["amd64"] = "6"
		dict["arm"] = "5"
		dict["compiler"] = "g"
		dict["linker"] = "l"
	}
	return dict[os.Getenv("GOARCH")] + dict[tool]
}

// Go compiler
func gocompile() string {
	return goexec("compiler")
}

// Go linker
func golink() string {
	return goexec("linker")
}

// Go architecture extension
func goext() string {
	return goexec("")
}

