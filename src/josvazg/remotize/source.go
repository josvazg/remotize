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

const (
	ProposedInterface = iota
	Type
	Interface
)

type candidate struct {
	status int
	value  interface{}
}

// remotize spec
type rinfo struct {
	currpack   string
	aliases    map[string]string
	methods    map[string][]*ast.FuncDecl
	candidates map[string]*candidate
	sources    map[string]string
	Imports    *bytes.Buffer
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
	parts := strings.Split(name, ".", -1)
	alias := r.aliases[parts[0]]
	if alias == "" {
		return name
	}
	return alias + "." + parts[1]
}
/*
func funcName(call *ast.CallExpr) string {
	se, ok := call.Fun.(*ast.SelectorExpr)
	name := ""
	if ok {
		if id, ok := se.X.(*ast.Ident); ok && id.Name == "remotize" {
			return se.Sel.Name
		}
	} else if r.currpack == "remotize" {
		if id, ok := call.Fun.(*ast.Ident);ok {
			return id.Name
		}
	}
	return ""
}*/

func markType(r *rinfo, name string) {
	typesrc := bytes.NewBufferString("type " + strings.TrimLeft(name, " *") +
		suffix(name) + " interface {")
	r.candidates[name] = &candidate{Type, typesrc}
	fmt.Println("Incomplete type from comments:", name)
}

// parseComment will search for interfaces or type with a comment 
// in the source code ended by '(remotize)' and will mark them for remotization
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
				r.candidates[name] = &candidate{Interface, it}
			} else {
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
			/*			str := "Suggested/Incomplete"
						if can.status == Incomplete {
							//can.status = Complete
							str = "Complete"
						}
						can.value = it
						fmt.Println(str+"candidate interface:", name)
					} else {*/
			r.candidates[name] = &candidate{ProposedInterface, it}
			fmt.Println("Proposed interface:", name)
		}
	}
}

// parseCalls will detect invocations of remotize calls like NewServer,
// NewClient or the empty marker RemotizePlease
func parseCalls(r *rinfo, call *ast.CallExpr) {
	if call.Fun == nil {
		return
	}
	name := solveName(call.Fun)
	var called string
	if name == "RemotizePlease" {
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
			if can.status == ProposedInterface {
				can.status = Interface
				fmt.Println("Interface from calls:", called,
					"value", can.value)
			}
		} else {
			/*r.candidates[called] = &candidate{Candidate, nil}
			fmt.Println("Type or Interface Candidate from calls:", called)*/
			markType(r, called)
		}
	}
}

// parseMethods will search for Function Declaration for types detected and 
// marked by parseComment 
func parseMethods(r *rinfo, fdecl *ast.FuncDecl) {
	if fdecl.Recv == nil {
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
	}
	return r
}

func postProcess(r *rinfo) {
	for name, can := range r.candidates { // complete types with methods
		if can.status == Type && can.value != nil {
			src := can.value.(*bytes.Buffer)
			if r.methods[name] != nil {
				for _, fdecl := range r.methods[name] {
					method := solveName(fdecl.Name)
					tmpbuf := bytes.NewBufferString("")
					printer.Fprint(tmpbuf, token.NewFileSet(), fdecl.Type)
					signature := tmpbuf.String()[4:]
					fmt.Fprintf(src, "\n\t%s%s", method, signature)
				}
				fmt.Fprintf(src, "\n}\n")
				r.sources[name] = src.String()
			}
		}
	}
	for name, can := range r.candidates {
		if can.status == Interface || can.status == Type {
			if it, ok := can.value.(*ast.InterfaceType); ok {
				src := bytes.NewBufferString("type " + name + " ")
				printer.Fprint(src, token.NewFileSet(), it)
				fmt.Fprintf(src, "\n")
				r.sources[name] = src.String()
			}
		}
	}
}

// parseImports will process imports for detection on each file's source code
func parseImports(r *rinfo, file *ast.File) {
	r.aliases = make(map[string]string)
	for _, decl := range file.Decls {
		imp, ok := (interface{})(decl).(*ast.ImportSpec)
		if !ok || imp.Name == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, "\"")
		if strings.Contains(path, "/") {
			parts := strings.Split(path, "/", -1)
			path = parts[len(parts)-1]
		}
		r.aliases[path] = imp.Name.Name
	}
}

// parseFile will process a go source file to detect interfaces to be remotized
func parseFiles(r *rinfo, files ...string) os.Error {
	var fs []*ast.File
	for _, f := range files {
		file, e := parser.ParseFile(token.NewFileSet(), f, nil,
			parser.ParseComments)
		if e != nil {
			return e
		}
		fs = append(fs, file)
		if r.currpack == "" {
			r.currpack = file.Name.Name
		} else if r.currpack != file.Name.Name {
			panic("One package at a time! (can't remotize files from " +
				r.currpack + " and " + file.Name.Name + " at the same time)")
		}
		parseImports(r, file)
		ast.Walk(r, file)
		//ast.Print(token.NewFileSet(), file)
	}
	postProcess(r)
	return nil
}

/*
	Autoremotize will remotize all interfaces that either:
		- Are defined with a comment including '(remotize)' at the end
		- Are used within remotize.NewClient(), NewServer() or PleaseRemotize() Calls 
*/
func Autoremotize(files ...string) (int, os.Error) {
	done := 0
	rs := &rinfo{}
	rs.candidates = make(map[string]*candidate)
	rs.methods = make(map[string][]*ast.FuncDecl)
	rs.sources = make(map[string]string)
	if e := parseFiles(rs, files...); e != nil {
		return 0, e
	}
	items := len(rs.sources)
	if items == 0 {
		fmt.Println("No 'remotizables' found")
		return done, nil
	}
	fmt.Printf("Found %v interfaces/types to remotize:\n", items)
	for name, src := range rs.sources {
		fmt.Printf("%v:\n%v", name, src)
	}
	/*e := build(rs)
	if e != nil {
		fmt.Println("Error:", e)
	}*/
	return done, nil
}

// Given an ast node, it tries to solve its name
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

// source interface specification
type srcIfaceSpec struct {
	name string
	pack string
	*ast.InterfaceType
}

// sortable fields implement sort.Interface
type SortableFields struct {
	f []*ast.Field
}

func (sf *SortableFields) Len() int {
	return len(sf.f)
}

func (sf *SortableFields) Less(i, j int) bool {
	return solveName(sf.f[i].Names[0]) < solveName(sf.f[j].Names[0])
}

func (sf *SortableFields) Swap(i, j int) {
	f := sf.f[i]
	sf.f[i] = sf.f[j]
	sf.f[j] = f
}

// newSrcIfaceSpec generates a source interface specification from source code
func newSrcIfaceSpec(name, pack string, it *ast.InterfaceType) *srcIfaceSpec {
	sis := &srcIfaceSpec{name, pack, it}
	sf := &SortableFields{it.Methods.List}
	sort.Sort(sf)
	return sis
}

func (is *srcIfaceSpec) Name() string {
	return is.name
}

func (is *srcIfaceSpec) PkgPath() string {
	return is.pack
}

func (is *srcIfaceSpec) NumMethod() int {
	return len(is.Methods.List)
}

/*
func (is *srcIfaceSpec) MethodSpec(i int) methodSpec {
	m := is.Methods.List[i]
	return &srcMethodSpec{solveName(m.Names[0]), (m.Type).(*ast.FuncType)}
}
*/

// source method specification
type srcMethodSpec struct {
	name string
	*ast.FuncType
}

func (m *srcMethodSpec) MethodName() string {
	return m.name
}

func (m *srcMethodSpec) NumIn() int {
	return len(m.Params.List)
}

func (m *srcMethodSpec) InName(i int) string {
	return solveName(m.Params.List[i])
}

func (m *srcMethodSpec) InElem(i int) string {
	s := m.InName(i)
	if strings.Index(s, "*") == 0 {
		return s[1:]
	}
	return s
}

func (m *srcMethodSpec) InPkg(i int) string {
	s := m.InName(i)
	if i := strings.Index(s, "."); i > 0 {
		return s[0:i]
	}
	return ""
}

func (m *srcMethodSpec) InIsPtr(i int) bool {
	s := m.InName(i)
	return strings.Index(s, "*") == 0
}

func (m *srcMethodSpec) NumOut() int {
	if m.Results == nil {
		return 0
	}
	return len(m.Results.List)
}

func (m *srcMethodSpec) OutName(i int) string {
	//ast.Print(token.NewFileSet(),m.Results.List[i])
	return solveName(m.Results.List[i])
}

func (m *srcMethodSpec) OutPkg(i int) string {
	s := m.OutName(i)
	if j := strings.Index(s, "."); j > 0 {
		return s[0:j]
	}
	return ""
}

func (m *srcMethodSpec) OutIsError(i int) bool {
	return m.OutName(i) == "os.Error"
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

