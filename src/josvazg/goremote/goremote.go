package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"reflect"
	"remotize"
	"strconv"
	"strings"
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
		if e := remotize0(f, header+decl); e != nil {
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
		if e := remotize0(f, header+decl); e != nil {
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
		if e := remotize0(f, source); e != nil {
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
		dcl = newDeclaration(t, "type "+st.Name()+remotize.Suffix(st.Name())+
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
		fmt.Fprintf(d.src, "["+strconv.Itoa(t.Len())+"]"+t.Name())
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
	buf := bytes.NewBufferString("package " + packname + "\n\n")
	d.imports["rpc"] = "rpc"
	d.imports["os"] = "os"
	if packname != "remotize" {
		d.imports["remotize"] = "remotize"
	}
	if len(d.imports) > 0 {
		fmt.Fprintf(buf, "import (\n")
		for _, i := range d.imports {
			v := d.imports[i]
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

func remotize0(w io.Writer, source string) os.Error {
	f := src2ast(source)
	//ast.Print(token.NewFileSet(), f)
	rprefix := "remotize."
	if f.Name.Name == "remotize" {
		rprefix = ""
	}
	for _, decl := range f.Decls {
		if gendecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gendecl.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if it, ok := ts.Type.(*ast.InterfaceType); ok {
						return remotizeInterface(w, rprefix, ts.Name.Name, it)
					}
				}
			}
		}
	}
	return nil
}

func remotizeInterface(out io.Writer, rprefix, ifacename string,
iface *ast.InterfaceType) os.Error {
	fmt.Fprintf(out, "// Autoregistry\n")
	fmt.Fprintf(out, "func init() {\n")
	fmt.Fprintf(out, "    %sRegister(Remote%s{},\n", rprefix, ifacename)
	fmt.Fprintf(out, "        func(cli *rpc.Client) interface{} "+
		"{\n\t\t\treturn NewRemote%s(cli)\n\t\t},\n", ifacename)
	fmt.Fprintf(out, "        %sService{},\n", ifacename)
	fmt.Fprintf(out, "        func(srv *rpc.Server, i interface{}) "+
		" interface{} {\n\t\t\treturn New%sService(srv,i.(%s))\n\t\t},\n",
		ifacename, ifacename)
	fmt.Fprintf(out, "    )\n")
	fmt.Fprintf(out, "}\n\n")
	remoteInit(out, ifacename)
	localInit(out, ifacename)
	for _, f := range iface.Methods.List {
		if ft, ok := f.Type.(*ast.FuncType); ok {
			wrapFunction(out, ifacename, f.Names[0].Name, ft)
		}
	}
	//fmt.Println(out.String())
	return nil
}

func remoteInit(out io.Writer, ifacename string) {
	fmt.Fprintf(out, "// Rpc service wrapper for %s\n", ifacename)
	fmt.Fprintf(out, "type %sService struct {\n", ifacename)
	fmt.Fprintf(out, "    srv %s\n", ifacename)
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "// Direct %sService constructor\n", ifacename)
	fmt.Fprintf(out, "func New%sService(srv *rpc.Server, impl %s) *%sService {\n",
		ifacename, ifacename, ifacename)
	fmt.Fprintf(out, "    r:=&%sService{impl}\n", ifacename)
	fmt.Fprintf(out, "    srv.Register(r)\n")
	fmt.Fprintf(out, "    return r\n")
	fmt.Fprintf(out, "}\n\n")
}

func localInit(out io.Writer, ifacename string) {
	fmt.Fprintf(out, "// Rpc client for %s\n", ifacename)
	fmt.Fprintf(out, "type Remote%s struct {\n", ifacename)
	fmt.Fprintf(out, "    cli *rpc.Client\n")
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "// Direct Remote%s constructor\n", ifacename)
	fmt.Fprintf(out, "func NewRemote%s(cli *rpc.Client) *Remote%s {\n",
		ifacename, ifacename)
	fmt.Fprintf(out, "    return &Remote%s{cli}\n", ifacename)
	fmt.Fprintf(out, "}\n\n")
}

func wrapFunction(out io.Writer, iface, name string, fun *ast.FuncType) {
	fmt.Fprintf(out, "// wrapper for: %s\n\n", name)
	argcnt := generateStructWrapper(out, fun.Params, "Args", name)
	results, inouts := prepareInOuts(fun.Params, fun.Results)
	replycnt := generateStructWrapper(out, results, "Reply", name)
	generateServerRPCWrapper(out, fun, iface, name, argcnt, replycnt, inouts)
	generateClientRPCWrapper(out, fun, iface, name, argcnt, replycnt, inouts)
	fmt.Fprintf(out, "\n")
}

func prepareInOuts(params *ast.FieldList, r *ast.FieldList) (*ast.FieldList,
[]int) {
	args := make([]*ast.Field, 0)
	inouts := make([]int, 0)
	for n, field := range params.List {
		if _, ok := field.Type.(*ast.StarExpr); ok {
			args = append(args, field)
			inouts = append(inouts, n)
		}
	}
	if r == nil {
		r = &ast.FieldList{List: args}
	} else {
		r.List = append(r.List, args...)
	}
	return r, inouts
}

func generateStructWrapper(out io.Writer, fun *ast.FieldList, structname,
name string) int {
	fmt.Fprintf(out, "type %s_%s struct {\n", structname, name)
	defer fmt.Fprintf(out, "}\n\n")
	argn := 0
	if fun == nil || len(fun.List) == 0 {
		return argn
	}
	for _, field := range fun.List {
		fmt.Fprintf(out, "\t")
		// names
		if field.Names != nil {
			for j, _ := range field.Names {
				fmt.Fprintf(out, "Arg%d", argn)
				if j != len(field.Names)-1 {
					fmt.Fprintf(out, ", ")
				}
				argn++
			}
			fmt.Fprintf(out, " ")
		} else {
			fmt.Fprintf(out, "Arg%d ", argn)
			argn++
		}

		// type
		prettyPrintTypeExpr(out, field.Type)

		// \n
		fmt.Fprintf(out, "\n")
	}
	return argn
}

func prettyPrintTypeExpr(out io.Writer, e ast.Expr) {
	ty := reflect.TypeOf(e)
	switch t := e.(type) {
	case *ast.StarExpr:
		fmt.Fprintf(out, "*")
		prettyPrintTypeExpr(out, t.X)
	case *ast.Ident:
		fmt.Fprintf(out, t.Name)
	case *ast.ArrayType:
		fmt.Fprintf(out, "[]")
		prettyPrintTypeExpr(out, t.Elt)
	case *ast.SelectorExpr:
		prettyPrintTypeExpr(out, t.X)
		fmt.Fprintf(out, ".%s", t.Sel.Name)
	case *ast.FuncType:
		fmt.Fprintf(out, "func(")
		prettyPrintFuncFieldList(out, t.Params)
		fmt.Fprintf(out, ")")

		buf := bytes.NewBuffer(make([]byte, 0, 256))
		nresults := prettyPrintFuncFieldList(buf, t.Results)
		if nresults > 0 {
			results := buf.String()
			if strings.Index(results, " ") != -1 {
				results = "(" + results + ")"
			}
			fmt.Fprintf(out, " %s", results)
		}
	case *ast.MapType:
		fmt.Fprintf(out, "map[")
		prettyPrintTypeExpr(out, t.Key)
		fmt.Fprintf(out, "]")
		prettyPrintTypeExpr(out, t.Value)
	case *ast.InterfaceType:
		fmt.Fprintf(out, "interface{}")
	case *ast.Ellipsis:
		fmt.Fprintf(out, "...")
		prettyPrintTypeExpr(out, t.Elt)
	default:
		fmt.Fprintf(out, "\n[!!] unknown type: %s\n", ty.String())
	}
}

func prettyPrintFuncFieldList(out io.Writer, f *ast.FieldList) int {
	count := 0
	if f == nil {
		return count
	}
	for i, field := range f.List {
		// names
		if field.Names != nil {
			for j, name := range field.Names {
				fmt.Fprintf(out, "%s", name.Name)
				if j != len(field.Names)-1 {
					fmt.Fprintf(out, ", ")
				}
				count++
			}
			fmt.Fprintf(out, " ")
		} else {
			count++
		}

		// type
		prettyPrintTypeExpr(out, field.Type)

		// ,
		if i != len(f.List)-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	return count
}

// function that is being exposed to an RPC API, but calls simple "Server_" one
func generateServerRPCWrapper(out io.Writer, fun *ast.FuncType,
iface, name string, argcnt, replycnt int, inouts []int) {
	fmt.Fprintf(out, "func (r *%sService) %s(args *Args_%s, "+
		"reply *Reply_%s) os.Error {\n", iface, name, name, name)

	fmt.Fprintf(out, "\t")
	replies := replycnt - len(inouts)
	for i := 0; i < replies; i++ {
		fmt.Fprintf(out, "reply.Arg%d", i)
		if i != replycnt-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	if replies > 0 {
		fmt.Fprintf(out, " = ")
	}
	fmt.Fprintf(out, "r.srv.%s(", name)
	for i := 0; i < argcnt; i++ {
		fmt.Fprintf(out, "args.Arg%d", i)
		if i != argcnt-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	fmt.Fprintf(out, ")\n")
	for i := replies; i < replycnt; i++ {
		fmt.Fprintf(out, "\treply.Arg%d=args.Arg%d\n", i, inouts[i-replies])
	}
	fmt.Fprintf(out, "\treturn nil\n}\n\n")
}

func generateClientRPCWrapper(out io.Writer, fun *ast.FuncType, iface,
name string, argcnt, replycnt int, inouts []int) {
	fmt.Fprintf(out, "func (l *Remote%s) %s(", iface, name)
	prettyPrintFuncFieldListUsingArgs(out, fun.Params)
	fmt.Fprintf(out, ")")

	buf := bytes.NewBuffer(make([]byte, 0, 256))
	nresults := prettyPrintFuncFieldList(buf, fun.Results)
	if nresults > 0 {
		results := buf.String()
		if strings.Index(results, " ") != -1 {
			results = "(" + results + ")"
		}
		fmt.Fprintf(out, " %s", results)
	}
	fmt.Fprintf(out, " {\n")
	fmt.Fprintf(out, "\tvar args Args_%s\n", name)
	fmt.Fprintf(out, "\tvar reply Reply_%s\n", name)
	for i := 0; i < argcnt; i++ {
		fmt.Fprintf(out, "\targs.Arg%d = Arg%d\n", i, i)
	}
	fmt.Fprintf(out, "\terr := l.cli.Call(\"%sService.%s\", &args, &reply)\n",
		iface, name)
	fmt.Fprintf(out, "\tif err != nil {\n")
	fmt.Fprintf(out, "\t\tpanic(err.String())\n\t}\n")

	replies := replycnt - len(inouts)
	for i := replies; i < replycnt; i++ {
		fmt.Fprintf(out, "\t*Arg%d=*reply.Arg%d\n", i, inouts[i-replies])
	}
	fmt.Fprintf(out, "\treturn ")
	for i := 0; i < replies; i++ {
		fmt.Fprintf(out, "reply.Arg%d", i)
		if i != replycnt-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	fmt.Fprintf(out, "\n}\n\n")
}

func prettyPrintFuncFieldListUsingArgs(out io.Writer, f *ast.FieldList) int {
	count := 0
	if f == nil {
		return count
	}
	for i, field := range f.List {
		// names
		if field.Names != nil {
			for j, _ := range field.Names {
				fmt.Fprintf(out, "Arg%d", count)
				if j != len(field.Names)-1 {
					fmt.Fprintf(out, ", ")
				}
				count++
			}
			fmt.Fprintf(out, " ")
		} else {
			fmt.Fprintf(out, "Arg%d ", count)
			count++
		}

		// type
		prettyPrintTypeExpr(out, field.Type)

		// ,
		if i != len(f.List)-1 {
			fmt.Fprintf(out, ", ")
		}
	}
	return count
}

// isExported returns true if the given FuncDecl is Exported (=Title-case)
func isExported(name string) bool {
	return name != "" && name[0:1] == strings.ToUpper(name[0:1])
}

// Main invoked Autoremotize()
func main() {
	flag.Parse()
	Autoremotize(flag.Args()...)
}

