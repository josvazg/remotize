package codegen

type GoFile struct {
	filename	string
	packname	string
	imports		[]importSpec
	constants	[]constantSpec
	variables	[]varSpec
	funcs		[]funcSpec
	types		[]typeSpec
	methods		[]methodSpec
}

type importSpec struct {
	name, alias string
}

type constantSpec struct {
	isenum 		bool
	name 		string
	value		interface{}
	names		[]string
	values		[]interface{}
}

type varSpec struct {
	name 	string
	value	[]interface{}
}

type funcSpec struct {
	name 		string
	argsNames	[]string
	args		[]interface{}
	resultNames	[]string
	code		[]codeElement
}

type codeElement struct {
}

type typeSpec struct {
	name string
}

type methodSpec struct {
	name string
}

func NewGoFile(name, pack string) (*GoFile) {
	return &GoFile{filename:name,packname:pack}
}

func (gf *GoFile) AddImport(name string) {
	gf.imports=append(gf.imports,importSpec{name,""})
}

func (gf *GoFile) AddAliasedImport(name, alias string) {
	gf.imports=append(gf.imports,importSpec{name,alias})
}

