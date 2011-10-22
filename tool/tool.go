// Copyright 2011 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// remotize package wraps rpc calls so you don't have to rewrite an interface by
// hand in order to use it remotely. 
package tool

import (
	"bytes"
	"exec"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"reflect"
	"github.com/josvazg/remotize/misc"
	"sort"
	"strconv"
	"strings"
)

// Candidate States
const (
	ProposedInterface = iota
	Type
	Interface
)

const RemotizePkg = "github.com/josvazg/remotize"

// remotizer code head and tail & marker
const (
	remotizerHead   = `// Autogenerated Remotizer [DO NOT EDIT!]
package main
`
	remotizerTail   = `func main() {
	for _,r := range toremotize {
		if e:=tool.Remotize(r); e!=nil {
			panic(e)
		}
	}
}`
	redefinedMarker = "\n// Redefined\n"
	specSeparator   = ":\n"
)

// startsWith returns true if str starts with substring s
func startsWith(str, s string) bool {
	return len(str) >= len(s) && str[:len(s)] == s
}

// endsWith returns true if str ends with substring s
func endsWith(str, s string) bool {
	return len(str) >= len(s) && str[len(str)-len(s):] == s
}

// splitSource will split the name and source code
func splitSource(spec string) (string, string) {
	parts := strings.SplitN(spec, specSeparator, 2)
	return parts[0], parts[1]
}

// Remotize remotizes a type, interface or source code specified interface
func Remotize(i interface{}) os.Error {
	var t reflect.Type
	if _, ok := i.(reflect.Type); ok {
		t = i.(reflect.Type)
	} else {
		t = reflect.TypeOf(i)
	}
	if t.Kind() == reflect.Interface {
		f, e := os.Create("remotized" + t.Name() + ".go")
		if e != nil {
			return e
		}
		header, decl := declare(t)
		fmt.Fprintf(f, header)
		if e := doremotize(t, f, header+decl); e != nil {
			return e
		}
		f.Close()
		return nil
	} else if t.NumMethod() > 0 {
		st := t
		for ; st.Kind() == reflect.Ptr; st = st.Elem() {
		}
		f, e := os.Create("remotized" + st.Name() + ".go")
		if e != nil {
			return e
		}
		header, decl := declare(t)
		fmt.Fprintf(f, header+decl)
		if e := doremotize(t, f, header+decl); e != nil {
			return e
		}
		f.Close()
		return nil
	}
	if t.Kind() == reflect.Ptr {
		return Remotize(t.Elem())
	}
	if t.Kind() == reflect.String {
		name, source := splitSource(i.(string))
		f, e := os.Create("remotized" + name + ".go")
		if e != nil {
			return e
		}
		fmt.Fprintln(f, source)
		if e := doremotize(t, f, source); e != nil {
			return e
		}
		f.Close()
		return nil
	}
	// TODO error
	return nil
}

func save(filename, contents string) {
	f, e := os.Create(filename)
	if e != nil {
		panic(e)
	}
	f.Write(([]byte)(contents))
	f.Close()
}

type declaration struct {
	t       reflect.Type
	src     *bytes.Buffer
	imports map[string]string
}

func declare(t reflect.Type) (header string, decl string) {
	var dcl *declaration
	switch t.Kind() {
	case reflect.Interface:
		dcl = newDeclaration(t, "type "+t.Name()+" interface")
	default:
		st := t
		for ; st.Kind() == reflect.Ptr; st = t.Elem() {
		}
		dcl = newDeclaration(t, "type "+st.Name()+misc.Suffix(st.Name())+
			" interface")
	}
	dcl.methods(t)
	return dcl.header(), dcl.src.String()
}

func newDeclaration(t reflect.Type, header string) *declaration {
	return &declaration{t, bytes.NewBufferString(header),
		make(map[string]string)}
}

func (d *declaration) methods(t reflect.Type) {
	if t.NumMethod() > 0 {
		fmt.Fprintf(d.src, " {")
		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)
			if isExported(m.Name) {
				fmt.Fprintf(d.src, "\n    ")
				d.funcsource(t, &m)
			}
		}
		fmt.Fprintf(d.src, "\n}\n\n")
	}
}

func (d *declaration) funcsource(t reflect.Type, m *reflect.Method) {
	start := 0
	if t.Kind() == reflect.Interface {
		fmt.Fprintf(d.src, m.Name+"(")
	} else if m != nil && m.Name != "" {
		start++
		fmt.Fprintf(d.src, m.Name+"(")
	} else {
		fmt.Fprintf(d.src, "func (")
	}
	if m != nil {
		t = m.Type
	}
	for i := start; i < t.NumIn(); i++ {
		d.source(t.In(i))
		if (i + 1) != t.NumIn() {
			fmt.Fprintf(d.src, ", ")
		}
	}
	fmt.Fprintf(d.src, ") ")
	if t.NumOut() > 1 {
		fmt.Fprintf(d.src, "(")
	}
	for i := 0; i < t.NumOut(); i++ {
		d.source(t.Out(i))
		if (i + 1) != t.NumOut() {
			fmt.Fprintf(d.src, ", ")
		}
	}
	if t.NumOut() > 1 {
		fmt.Fprintf(d.src, ")")
	}
}

func (d *declaration) source(t reflect.Type) {
	switch t.Kind() {
	case reflect.Array:
		fmt.Fprintf(d.src, "["+strconv.Itoa(t.Len())+"]")
		d.source(t.Elem())
	case reflect.Chan:
		fmt.Fprintf(d.src, "chan ")
		d.source(t.Elem())
	case reflect.Func:
		d.funcsource(t, nil)
	case reflect.Map:
		fmt.Fprintf(d.src, "map[")
		d.source(t.Key())
		fmt.Fprintf(d.src, "]")
		d.source(t.Elem())
	case reflect.Ptr:
		fmt.Fprintf(d.src, "*")
		d.source(t.Elem())
	case reflect.Slice:
		fmt.Fprintf(d.src, "[]")
		d.source(t.Elem())
	case reflect.String:
		fmt.Fprintf(d.src, "string")
	default:
		d.pack(t)
		fmt.Fprintf(d.src, t.String())
		return
	}
}

func (d *declaration) pack(t reflect.Type) {
	packpath := t.PkgPath()
	if packpath != "" {
		alias := t.String()[0 : len(t.String())-len(t.Name())-1]
		d.imports[alias] = packpath
	}
}

func packname(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return packname(t.Elem())
	}
	path := t.PkgPath()
	parts := strings.Split(path, "/")
	if parts == nil && len(parts) == 1 {
		return path
	}
	return parts[len(parts)-1]
}

func (d *declaration) header() string {
	packname := packname(d.t)
	buf := bytes.NewBufferString("// Autogenerated by josvazg/remotize/tool - no need to edit!\n\n")
	fmt.Fprintf(buf, "package %v\n\n", packname)
	d.imports["rpc"] = "rpc"
	d.imports["os"] = "os"
	if packname != RemotizePkg {
		d.imports[RemotizePkg] = RemotizePkg
	}
	if len(d.imports) > 0 {
		fmt.Fprintf(buf, "import (\n")
		for _, i := range d.imports {
			v := d.imports[i]
			if v == packname || v == "" {
				continue
			}
			if i == v {
				fmt.Fprintf(buf, "    \"%v\"\n", i)
			} else {
				fmt.Fprintf(buf, "    %v \"%v\"\n", v, i)
			}
		}
		fmt.Fprintf(buf, ")\n\n")
	}
	return buf.String()
}

func src2ast(src string) *ast.File {
	//fmt.Println(src)
	f, e := parser.ParseFile(token.NewFileSet(), "", src, 0)
	if e != nil {
		panic(e.String() + ":\n" + src)
	}
	return f
}

type remotizeCtx struct {
	pkg string
	w   io.Writer
}

func doremotize(t reflect.Type, w io.Writer, source string) os.Error {
	rctx := remotizeCtx{packname(t), w}
	f := src2ast(source)
	//ast.Print(token.NewFileSet(), f)
	rprefix := "remotize."
	if f.Name.Name == RemotizePkg {
		rprefix = ""
	}
	for _, decl := range f.Decls {
		if gendecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gendecl.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if it, ok := ts.Type.(*ast.InterfaceType); ok {
						return rctx.remotizeInterface(rprefix, ts.Name.Name, it)
					}
				}
			}
		}
	}
	return nil
}

func (r *remotizeCtx) remotizeInterface(rprefix, ifacename string,
iface *ast.InterfaceType) os.Error {
	fmt.Fprintf(r.w, "// Autoregistry\n")
	fmt.Fprintf(r.w, "func init() {\n")
	fmt.Fprintf(r.w, "    %sRegister(Remote%s{},\n", rprefix, ifacename)
	fmt.Fprintf(r.w, "        func(cli *rpc.Client) interface{} "+
		"{\n\t\t\treturn NewRemote%s(cli)\n\t\t},\n", ifacename)
	fmt.Fprintf(r.w, "        %sService{},\n", ifacename)
	fmt.Fprintf(r.w, "        func(srv *rpc.Server, i interface{}) "+
		" interface{} {\n\t\t\treturn New%sService(srv,i.(%s))\n\t\t},\n",
		ifacename, ifacename)
	fmt.Fprintf(r.w, "    )\n")
	fmt.Fprintf(r.w, "}\n\n")
	r.remoteInit(ifacename)
	r.localInit(ifacename)
	for _, f := range iface.Methods.List {
		if ft, ok := f.Type.(*ast.FuncType); ok {
			r.wrapFunction(ifacename, f.Names[0].Name, ft)
		}
	}
	//fmt.Println(r.w.String())
	return nil
}

func (r *remotizeCtx) remoteInit(ifacename string) {
	fmt.Fprintf(r.w, "// Rpc service wrapper for %s\n", ifacename)
	fmt.Fprintf(r.w, "type %sService struct {\n", ifacename)
	fmt.Fprintf(r.w, "    srv %s\n", ifacename)
	fmt.Fprintf(r.w, "}\n\n")
	fmt.Fprintf(r.w, "// Direct %sService constructor\n", ifacename)
	fmt.Fprintf(r.w, "func New%sService(srv *rpc.Server, impl %s) *%sService {\n",
		ifacename, ifacename, ifacename)
	fmt.Fprintf(r.w, "    r:=&%sService{impl}\n", ifacename)
	fmt.Fprintf(r.w, "    srv.Register(r)\n")
	fmt.Fprintf(r.w, "    return r\n")
	fmt.Fprintf(r.w, "}\n\n")
}

func (r *remotizeCtx) localInit(ifacename string) {
	fmt.Fprintf(r.w, "// Rpc client for %s\n", ifacename)
	fmt.Fprintf(r.w, "type Remote%s struct {\n", ifacename)
	fmt.Fprintf(r.w, "    cli *rpc.Client\n")
	fmt.Fprintf(r.w, "}\n\n")
	fmt.Fprintf(r.w, "// Direct Remote%s constructor\n", ifacename)
	fmt.Fprintf(r.w, "func NewRemote%s(cli *rpc.Client) *Remote%s {\n",
		ifacename, ifacename)
	fmt.Fprintf(r.w, "    return &Remote%s{cli}\n", ifacename)
	fmt.Fprintf(r.w, "}\n\n")
}

func (r *remotizeCtx) wrapFunction(iface, name string, fun *ast.FuncType) {
	fmt.Fprintf(r.w, "// wrapper for: %s\n\n", name)
	argcnt := r.generateStructWrapper(fun.Params, "Args", name)
	results, inouts := r.prepareInOuts(fun.Params, fun.Results)
	replycnt := r.generateStructWrapper(results, "Reply", name)
	r.generateServerRPCWrapper(fun, iface, name, argcnt, replycnt, inouts)
	r.generateClientRPCWrapper(fun, iface, name, argcnt, replycnt, inouts)
	fmt.Fprintf(r.w, "\n")
}

func (r *remotizeCtx) prepareInOuts(params *ast.FieldList,
results *ast.FieldList) (*ast.FieldList, []int) {
	args := make([]*ast.Field, 0)
	inouts := make([]int, 0)
	for n, field := range params.List {
		if _, ok := field.Type.(*ast.StarExpr); ok {
			args = append(args, field)
			inouts = append(inouts, n)
		}
	}
	if results == nil {
		results = &ast.FieldList{List: args}
	} else {
		results.List = append(results.List, args...)
	}
	return results, inouts
}

func (r *remotizeCtx) generateStructWrapper(fun *ast.FieldList, structname,
name string) int {
	fmt.Fprintf(r.w, "type %s_%s struct {\n", structname, name)
	defer fmt.Fprintf(r.w, "}\n\n")
	argn := 0
	if fun == nil || len(fun.List) == 0 {
		return argn
	}
	for _, field := range fun.List {
		fmt.Fprintf(r.w, "\t")
		// names
		if field.Names != nil {
			for j, _ := range field.Names {
				fmt.Fprintf(r.w, "Arg%d", argn)
				if j != len(field.Names)-1 {
					fmt.Fprintf(r.w, ", ")
				}
				argn++
			}
			fmt.Fprintf(r.w, " ")
		} else {
			fmt.Fprintf(r.w, "Arg%d ", argn)
			argn++
		}

		// type
		r.printTypeExpr(r.w, field.Type)

		// \n
		fmt.Fprintf(r.w, "\n")
	}
	return argn
}

func (r *remotizeCtx) printTypeExpr(w io.Writer, e ast.Expr) {
	ty := reflect.TypeOf(e)
	switch t := e.(type) {
	case *ast.StarExpr:
		fmt.Fprintf(w, "*")
		r.printTypeExpr(w, t.X)
	case *ast.Ident:
		fmt.Fprintf(w, t.Name)
	case *ast.ArrayType:
		fmt.Fprintf(w, "[%v]", solveName(t.Len))
		r.printTypeExpr(w, t.Elt)
	case *ast.SelectorExpr:
		buf := bytes.NewBuffer(make([]byte, 0, 256))
		r.printTypeExpr(buf, t.X)
		prefix := buf.String()
		if prefix != r.pkg {
			fmt.Fprintf(w, "%s.", prefix)
		}
		fmt.Fprintf(w, "%s", t.Sel.Name)
	case *ast.FuncType:
		fmt.Fprintf(w, "func(")
		r.printFuncFieldList(w, t.Params)
		fmt.Fprintf(w, ")")
		buf := bytes.NewBuffer(make([]byte, 0, 512))
		nresults := r.printFuncFieldList(buf, t.Results)
		if nresults > 0 {
			results := buf.String()
			if strings.Index(results, " ") != -1 {
				results = "(" + results + ")"
			}
			fmt.Fprintf(w, " %s", results)
		}
	case *ast.MapType:
		fmt.Fprintf(w, "map[")
		r.printTypeExpr(w, t.Key)
		fmt.Fprintf(w, "]")
		r.printTypeExpr(w, t.Value)
	case *ast.InterfaceType:
		fmt.Fprintf(w, "interface{}")
	case *ast.Ellipsis:
		fmt.Fprintf(w, "...")
		r.printTypeExpr(w, t.Elt)
	default:
		fmt.Fprintf(w, "\n[!!] unknown type: %s\n", ty.String())
	}
}

func (r *remotizeCtx) printFuncFieldList(w io.Writer,
f *ast.FieldList) int {
	count := 0
	if f == nil {
		return count
	}
	for i, field := range f.List {
		// names
		if field.Names != nil {
			for j, name := range field.Names {
				fmt.Fprintf(w, "%s", name.Name)
				if j != len(field.Names)-1 {
					fmt.Fprintf(w, ", ")
				}
				count++
			}
			fmt.Fprintf(w, " ")
		} else {
			count++
		}

		// type
		r.printTypeExpr(w, field.Type)

		// ,
		if i != len(f.List)-1 {
			fmt.Fprintf(w, ", ")
		}
	}
	return count
}

// function that is valeing exposed to an RPC API, but calls simple "Server_" one
func (r *remotizeCtx) generateServerRPCWrapper(fun *ast.FuncType,
iface, name string, argcnt, replycnt int, inouts []int) {
	fmt.Fprintf(r.w, "func (r *%sService) %s(args *Args_%s, "+
		"reply *Reply_%s) os.Error {\n", iface, name, name, name)

	fmt.Fprintf(r.w, "\t")
	replies := replycnt - len(inouts)
	for i := 0; i < replies; i++ {
		fmt.Fprintf(r.w, "reply.Arg%d", i)
		if i != replies-1 {
			fmt.Fprintf(r.w, ", ")
		}
	}
	if replies > 0 {
		fmt.Fprintf(r.w, " = ")
	}
	fmt.Fprintf(r.w, "r.srv.%s(", name)
	for i := 0; i < argcnt; i++ {
		fmt.Fprintf(r.w, "args.Arg%d", i)
		if i != argcnt-1 {
			fmt.Fprintf(r.w, ", ")
		}
	}
	fmt.Fprintf(r.w, ")\n")
	for i := replies; i < replycnt; i++ {
		fmt.Fprintf(r.w, "\treply.Arg%d=args.Arg%d\n", i, inouts[i-replies])
	}
	fmt.Fprintf(r.w, "\treturn nil\n}\n\n")
}

func (r *remotizeCtx) generateClientRPCWrapper(fun *ast.FuncType, iface,
name string, argcnt, replycnt int, inouts []int) {
	fmt.Fprintf(r.w, "func (l *Remote%s) %s(", iface, name)
	r.printFuncFieldListUsingArgs(fun.Params)
	fmt.Fprintf(r.w, ")")

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	nresults := r.printFuncFieldList(buf, fun.Results)
	if nresults > 0 {
		results := buf.String()
		if strings.Index(results, " ") != -1 {
			results = "(" + results + ")"
		}
		fmt.Fprintf(r.w, " %s", results)
	}
	fmt.Fprintf(r.w, " {\n")
	fmt.Fprintf(r.w, "\tvar args Args_%s\n", name)
	fmt.Fprintf(r.w, "\tvar reply Reply_%s\n", name)
	for i := 0; i < argcnt; i++ {
		fmt.Fprintf(r.w, "\targs.Arg%d = Arg%d\n", i, i)
	}
	fmt.Fprintf(r.w, "\terr := l.cli.Call(\"%sService.%s\", &args, &reply)\n",
		iface, name)
	fmt.Fprintf(r.w, "\tif err != nil {\n")
	fmt.Fprintf(r.w, "\t\tpanic(err.String())\n\t}\n")

	replies := replycnt - len(inouts)
	for i := replies; i < replycnt; i++ {
		fmt.Fprintf(r.w, "\t*reply.Arg%d=*args.Arg%d\n", i, inouts[i-replies])
	}
	fmt.Fprintf(r.w, "\treturn ")
	for i := 0; i < replycnt; i++ {
		fmt.Fprintf(r.w, "reply.Arg%d", i)
		if i != replycnt-1 {
			fmt.Fprintf(r.w, ", ")
		}
	}
	fmt.Fprintf(r.w, "\n}\n\n")
}

func (r *remotizeCtx) printFuncFieldListUsingArgs(f *ast.FieldList) int {
	count := 0
	if f == nil {
		return count
	}
	for i, field := range f.List {
		// names
		if field.Names != nil {
			for j, _ := range field.Names {
				fmt.Fprintf(r.w, "Arg%d", count)
				if j != len(field.Names)-1 {
					fmt.Fprintf(r.w, ", ")
				}
				count++
			}
			fmt.Fprintf(r.w, " ")
		} else {
			fmt.Fprintf(r.w, "Arg%d ", count)
			count++
		}

		// type
		r.printTypeExpr(r.w, field.Type)

		// ,
		if i != len(f.List)-1 {
			fmt.Fprintf(r.w, ", ")
		}
	}
	return count
}

// isExported returns true if the given FuncDecl is Exported (=Title-case)
func isExported(name string) bool {
	return name != "" && name[0:1] == strings.ToUpper(name[0:1])
}

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
	return strings.TrimLeft(name, " *") + misc.Suffix(name)
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
			//fmt.Println("Purposed interface:", name)
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
	// fmt.Println("Recorded method for", recv, ":", *fdecl)
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
		//fmt.Println("import aliases", r.aliases)
	} else if name != path {
		r.aliases[name] = path
		//fmt.Println("* import aliases", r.aliases)
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
	//fmt.Println("About to parse ", files)
	for _, f := range files {
		//fmt.Println("Parsing ", f, "?") 
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
		//fmt.Println("Parsing ", f, "...")
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
	imports := []string{"remotize/tool"}
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
	if o, e := RunCmd(Gocompile(), "-I", "_test", filename+".go"); e != nil {
		fmt.Fprintf(os.Stderr, string(o)+"\n")
		return e
	}
	if o, e := RunCmd(Golink(), "-L", "_test", "-o", filename,
		filename+"."+Goext()); e != nil {
		fmt.Fprintf(os.Stderr, string(o)+"\n")
		return e
	}
	if o, e := RunCmd("./" + filename); e != nil {
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

// runs a command
func RunCmd(cmdargs ...string) ([]byte, os.Error) {
	fmt.Println(cmdargs)
	return exec.Command(cmdargs[0], cmdargs[1:]...).CombinedOutput()
}

// dictionary cache
var dict map[string]string

// go tool execution string
func Goexec(tool string) string {
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
func Gocompile() string {
	return Goexec("compiler")
}

// Go linker
func Golink() string {
	return Goexec("linker")
}

// Go architecture extension
func Goext() string {
	return Goexec("")
}
