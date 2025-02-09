package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"
	"gopkg.in/errgo.v1"
)

// TODO:
// - generate exported types if the parameter/response types aren't exported?
// - deal with literal interface and struct types.
// - copy doc comments from server methods.

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: httprequest-generate server-package server-type client-type\n")
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 3 {
		flag.Usage()
	}

	serverPkg, serverType, clientType := flag.Arg(0), flag.Arg(1), flag.Arg(2)

	if err := generate(serverPkg, serverType, clientType); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

type templateArg struct {
	CommandLine string
	PkgName     string
	Imports     []string
	Methods     []method
	ClientType  string
}

var code = template.Must(template.New("").Parse(`
// The code in this file was automatically generated by running
// 	httprequest-generate-client {{.CommandLine}}
// DO NOT EDIT

package {{.PkgName}}
import (
	{{range .Imports}}{{printf "%q" .}}
	{{end}}
)

type {{.ClientType}} struct {
	Client httprequest.Client
}

{{range .Methods}}
{{if .RespType}}
	{{.Doc}}
	func (c *{{$.ClientType}}) {{.Name}}(p *{{.ParamType}}) ({{.RespType}}, error) {
		var r {{.RespType}}
		err := c.Client.Call(p, &r)
		return r, err
	}
{{else}}
	{{.Doc}}
	func (c *{{$.ClientType}}) {{.Name}}(p *{{.ParamType}}) (error) {
		return c.Client.Call(p, nil)
	}
{{end}}
{{end}}
`))

func generate(serverPkgPath, serverType, clientType string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}
	localPkg, err := build.Import(".", currentDir, 0)
	if err != nil {
		return errgo.Notef(err, "cannot open package in current directory")
	}
	serverPkg, err := build.Import(serverPkgPath, currentDir, 0)
	if err != nil {
		return errgo.Notef(err, "cannot open %q", serverPkgPath)
	}

	methods, imports, err := serverMethods(serverPkg.ImportPath, serverType, localPkg.ImportPath)
	if err != nil {
		return errgo.Mask(err)
	}
	arg := templateArg{
		CommandLine: strings.Join(flag.Args(), " "),
		Imports:     imports,
		Methods:     methods,
		PkgName:     localPkg.Name,
		ClientType:  clientType,
	}
	var buf bytes.Buffer
	if err := code.Execute(&buf, arg); err != nil {
		return errgo.Mask(err)
	}
	data, err := format.Source(buf.Bytes())
	if err != nil {
		return errgo.Notef(err, "cannot format source")
	}
	if err := writeOutput(data, clientType); err != nil {
		return errgo.Mask(err)
	}
	return nil
}

func writeOutput(data []byte, clientType string) error {
	filename := strings.ToLower(clientType) + "_generated.go"
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return errgo.Mask(err)
	}
	return nil
}

type method struct {
	Name      string
	Doc       string
	ParamType string
	RespType  string
}

func serverMethods(serverPkg, serverType, localPkg string) ([]method, []string, error) {
	cfg := loader.Config{
		TypeCheckFuncBodies: func(string) bool {
			return false
		},
		ImportPkgs: map[string]bool{
			serverPkg: false, // false means don't load tests.
		},
		ParserMode: parser.ParseComments,
	}
	prog, err := cfg.Load()
	if err != nil {
		return nil, nil, errgo.Notef(err, "cannot load %q", serverPkg)
	}
	pkgInfo := prog.Imported[serverPkg]
	if pkgInfo == nil {
		return nil, nil, errgo.Newf("cannot find %q in imported code", serverPkg)
	}
	pkg := pkgInfo.Pkg
	obj := pkg.Scope().Lookup(serverType)
	if obj == nil {
		return nil, nil, errgo.Newf("type %s not found in %s", serverType, serverPkg)
	}
	objTypeName, ok := obj.(*types.TypeName)
	if !ok {
		return nil, nil, errgo.Newf("%s is not a type", serverType)
	}
	// Use the pointer type to get as many methods as possible.
	ptrObjType := types.NewPointer(objTypeName.Type())

	imports := map[string]string{
		"github.com/juju/httprequest": "httprequest",
		localPkg:                      "",
	}
	var methods []method
	mset := types.NewMethodSet(ptrObjType)
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		if !sel.Obj().Exported() {
			continue
		}
		name := sel.Obj().Name()
		if name == "Close" {
			continue
		}
		ptype, rtype, err := parseMethodType(sel.Type().(*types.Signature))
		if err != nil {
			fmt.Fprintf(os.Stderr, "ignoring method %s: %v\n", name, err)
			continue
		}
		comment := docComment(prog, sel)
		methods = append(methods, method{
			Name:      name,
			Doc:       comment,
			ParamType: typeStr(ptype, imports),
			RespType:  typeStr(rtype, imports),
		})
	}
	delete(imports, localPkg)
	var allImports []string
	for path := range imports {
		allImports = append(allImports, path)
	}
	return methods, allImports, nil
}

// docComment returns the doc comment for the method referred to
// by the given selection.
func docComment(prog *loader.Program, sel *types.Selection) string {
	obj := sel.Obj()
	tokFile := prog.Fset.File(obj.Pos())
	if tokFile == nil {
		panic("no file found for method")
	}
	filename := tokFile.Name()
	for _, pkgInfo := range prog.AllPackages {
		for _, f := range pkgInfo.Files {
			if tokFile := prog.Fset.File(f.Pos()); tokFile == nil || tokFile.Name() != filename {
				continue
			}
			// We've found the file we're looking for. Now traverse all
			// top level declarations looking for the right function declaration.
			for _, decl := range f.Decls {
				fdecl, ok := decl.(*ast.FuncDecl)
				if ok && fdecl.Name.Pos() == obj.Pos() {
					// Found it!
					return commentStr(fdecl.Doc)
				}
			}
		}
	}
	panic("method declaration not found")
}

func commentStr(c *ast.CommentGroup) string {
	if c == nil {
		return ""
	}
	var b []byte
	for i, cc := range c.List {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, cc.Text...)
	}
	return string(b)
}

// typeStr returns the type string to be used when using the
// given type. It adds any needed import paths to the given
// imports map (map from package path to package id).
func typeStr(t types.Type, imports map[string]string) string {
	if t == nil {
		return ""
	}
	qualify := func(pkg *types.Package) string {
		if name, ok := imports[pkg.Path()]; ok {
			return name
		}
		name := pkg.Name()
		// Make sure we're not duplicating the name.
		// TODO if we are, make a new non-duplicated version.
		for oldPkg, oldName := range imports {
			if oldName == name {
				panic(errgo.Newf("duplicate package name %s vs %s", pkg.Path(), oldPkg))
			}
		}
		imports[pkg.Path()] = name
		return name
	}
	return types.TypeString(t, qualify)
}

func parseMethodType(t *types.Signature) (ptype, rtype types.Type, err error) {
	mp := t.Params()
	if mp.Len() != 1 && mp.Len() != 2 {
		return nil, nil, errgo.New("wrong argument count")
	}
	ptype0 := mp.At(mp.Len() - 1).Type()
	ptype1, ok := ptype0.(*types.Pointer)
	if !ok {
		return nil, nil, errgo.New("parameter is not a pointer")
	}
	ptype = ptype1.Elem()
	if _, ok := ptype.Underlying().(*types.Struct); !ok {
		return nil, nil, errgo.Newf("parameter is %s, not a pointer to struct", ptype1.Elem())
	}
	rp := t.Results()
	if rp.Len() > 2 {
		return nil, nil, errgo.New("wrong result count")
	}
	if rp.Len() == 2 {
		rtype = rp.At(0).Type()
	}
	return ptype, rtype, nil
}
