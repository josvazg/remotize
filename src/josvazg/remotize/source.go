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

// remotize spec
type rinfo struct {
	currpack string
	aliases  map[string]string
	sources  map[string]*bytes.Buffer
	pending  int
	Imports  *bytes.Buffer
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

// parseRemotizeCalls will detect invocations of remotize calls like NewServer,
// NewClient or the empty marker RemotizePlease
func parseRemotizeCalls(r *rinfo, decl ast.Decl) {
	call, ok := (interface{})(decl).(*ast.CallExpr)
	if !ok {
		return
	}
	if call.Fun == nil {
		return
	}
	se, ok := call.Fun.(*ast.SelectorExpr)
	name := ""
	if ok {
		id, ok := se.X.(*ast.Ident)
		if !ok || id.Name != "remotize" {
			return
		}
		name = se.Sel.Name
	} else if r.currpack == "remotize" {
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			return
		}
		name = id.Name
	}
	argpos := -1
	if name == "RemotizePlease" {
		argpos = 0
	}
	if name == "NewServer" || name == "NewClient" {
		argpos = 1
	}
	if len(call.Args) < (argpos+1) || call.Args[argpos] == nil {
		return
	}
	subcall, ok := call.Args[argpos].(*ast.CallExpr)
	if !ok {
		return
	}
	cn, ok := subcall.Fun.(*ast.Ident)
	if !ok || cn.Name != "new" {
		return
	}
	if len(subcall.Args) < 1 || subcall.Args[0] == nil {
		return
	}
	_, ok = subcall.Args[0].(*ast.Ident)
	if !ok {
		return
	}
	// accquire
}

// parseComment will search for interfaces or type with a comment 
// in the source code ended by '(remotize)' and will mark them for remotization
func parseComment(r *rinfo, idecl ast.Decl) {
	decl, ok := (interface{})(idecl).(*ast.GenDecl)
	if !ok {
		return
	}
	if decl.Doc == nil {
		return
	}
	if decl.Specs == nil || len(decl.Specs) == 0 {
		return
	}
	tspec, ok := decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return
	}
	name := tspec.Name.Name
	i := len(decl.Doc.List) - 1
	for ; i >= 0 && empty(decl.Doc.List[i].Text); i-- {
	}
	if i >= 0 {
		cmt := decl.Doc.List[i]
		c := string(cmt.Text)
		if strings.Contains(strings.ToLower(c), "(remotize)") {
			if _, ok := r.sources[name]; ok {
				return
			}
			if _, ok := tspec.Type.(*ast.InterfaceType); ok {
				r.sources[name] = bytes.NewBufferString("type ")
				printer.Fprint(r.sources[name], token.NewFileSet(), tspec)
			} else {
				r.sources[name] = bytes.NewBufferString("// for " + name +
					"\ntype ")
				fmt.Fprintf(r.sources[name], "%s%s interface {",
					name, suffix(name))
				r.sources["*"+name] = bytes.NewBufferString("// for *" + name +
					"\ntype ")
				fmt.Fprintf(r.sources["*"+name], "%s%s interface {",
					name, suffix(name))
				r.pending++
			}
		}
	}
}

// parseComment will search for interfaces or type with a comment 
// in the source code ended by '(remotize)' and will mark them for remotization
func parseMethods(r *rinfo, idecl ast.Decl) {
	fdecl, ok := (interface{})(idecl).(*ast.FuncDecl)
	if !ok || fdecl.Recv == nil {
		return
	}
	recv := solveName(fdecl.Recv.List[0])
	if r.sources[recv] != nil {
		method := solveName(fdecl.Name)
		tmpbuf := bytes.NewBufferString("")
		printer.Fprint(tmpbuf, token.NewFileSet(), fdecl.Type)
		signature := tmpbuf.String()[4:]
		fmt.Fprintf(r.sources[recv], "\n\t%s%s", method, signature)
	}
}

// parseRemotizeDemands detects comments or calls requiring to remotize some 
// interface
func parseRemotizeDemands(r *rinfo, file *ast.File) {
	for _, decl := range file.Decls {
		parseComment(r, decl)
		//parseRemotizeCalls(r, decl)
	}
}

func closeSources(r *rinfo) {
	if r.pending > 0 {
		for recv, src := range r.sources {
			s := src.String()
			endsWith := s[len(s)-1:]
			if endsWith == "{" {
				r.sources[recv] = nil, false
			} else if endsWith != "}" {
				fmt.Fprintf(r.sources[recv], "\n}\n")
			} else {
				fmt.Fprintf(r.sources[recv], "\n")
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
		}
	}
	for _, file := range fs {
		parseImports(r, file)
		parseRemotizeDemands(r, file)
		//ast.Print(token.NewFileSet(), file)		
	}
	for _, file := range fs {
		for _, decl := range file.Decls {
			parseMethods(r, decl)
		}
	}
	closeSources(r)
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
	rs.sources = make(map[string]*bytes.Buffer)
	if e := parseFiles(rs, files...); e != nil {
		return 0, e
	}
	items := len(rs.sources)
	if items == 0 {
		fmt.Println("No 'remotizables' found")
		return done, nil
	}
	fmt.Printf("Found %v interfaces to remotize:\n", items)
	for name, src := range rs.sources {
		fmt.Printf("%v:\n%v", name, src.String())
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

