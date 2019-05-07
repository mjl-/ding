package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/mjl-/sherpadoc"
)

// ParsedPackage possibly includes some of its imports because the package that contains the section references it.
type parsedPackage struct {
	Path    string       // Of import, used for keeping duplicate type names from different packages unique.
	Pkg     *ast.Package // Needed for its files: we need a file to find the package path and identifier used to reference other types.
	Docpkg  *doc.Package
	Imports map[string]*parsedPackage // Package/import path to parsed packages.
}

type typewords []string

func (pp *parsedPackage) lookupType(name string) *doc.Type {
	for _, t := range pp.Docpkg.Types {
		if t.Name == name {
			return t
		}
	}
	return nil
}

// Documentation for a single field, with text above the field, and
// on the right of the field combined.
func fieldDoc(f *ast.Field) string {
	s := ""
	if f.Doc != nil {
		s += strings.Replace(strings.TrimSpace(f.Doc.Text()), "\n", " ", -1)
	}
	if f.Comment != nil {
		if s != "" {
			s += "; "
		}
		s += strings.TrimSpace(f.Comment.Text())
	}
	return s
}

// Parse string literal. Errors are fatal.
func parseStringLiteral(s string) string {
	r, err := strconv.Unquote(s)
	check(err, "parsing string literal")
	return r
}

func jsonName(tag string, name string) string {
	s := reflect.StructTag(tag).Get("json")
	if s == "" || strings.HasPrefix(s, ",") {
		return name
	} else if s == "-" {
		return ""
	} else {
		return strings.Split(s, ",")[0]
	}
}

// Return the names (can be none) for a field. Takes exportedness
// and JSON tag annotation into account.
func nameList(names []*ast.Ident, tag *ast.BasicLit) []string {
	if names == nil {
		return nil
	}
	l := []string{}
	for _, name := range names {
		if ast.IsExported(name.Name) {
			l = append(l, name.Name)
		}
	}
	if len(l) == 1 && tag != nil {
		name := jsonName(parseStringLiteral(tag.Value), l[0])
		if name != "" {
			return []string{name}
		}
		return nil
	}
	return l
}

// Parses a top-level sherpadoc section.
func parseDoc(apiName, packagePath string) *section {
	fset := token.NewFileSet()
	pkgs, firstErr := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	check(firstErr, "parsing code")
	for _, pkg := range pkgs {
		docpkg := doc.New(pkg, "", doc.AllDecls)

		for _, t := range docpkg.Types {
			if t.Name == apiName {
				par := &parsedPackage{
					Path:    packagePath,
					Pkg:     pkg,
					Docpkg:  docpkg,
					Imports: make(map[string]*parsedPackage),
				}
				return parseSection(t, par)
			}
		}
	}
	log.Fatalf("type %q not found\n", apiName)
	return nil
}

// Parse a section and its optional subsections, recursively.
// t is the type of the struct with the sherpa methods to be parsed.
func parseSection(t *doc.Type, pp *parsedPackage) *section {
	sec := &section{
		t.Name,
		t.Name,
		strings.TrimSpace(t.Doc),
		nil,
		map[string]struct{}{},
		nil,
		nil,
	}

	// make list of methods to parse, sorted by position in file name.
	methods := make([]*doc.Func, len(t.Methods))
	copy(methods, t.Methods)
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Decl.Name.NamePos < methods[j].Decl.Name.NamePos
	})

	for _, fn := range methods {
		parseMethod(sec, fn, pp)
	}

	// parse subsections
	ts := t.Decl.Specs[0].(*ast.TypeSpec)
	expr := ts.Type
	st := expr.(*ast.StructType)
	for _, f := range st.Fields.List {
		ident, ok := f.Type.(*ast.Ident)
		if !ok {
			continue
		}
		name := ident.Name
		if f.Tag != nil {
			name = reflect.StructTag(parseStringLiteral(f.Tag.Value)).Get("sherpa")
		}
		subt := pp.lookupType(ident.Name)
		if subt == nil {
			log.Fatalf("subsection %q not found\n", ident.Name)
		}
		subsec := parseSection(subt, pp)
		subsec.Name = name
		sec.Sections = append(sec.Sections, subsec)
	}
	return sec
}

// Ensure type "t" - used in a field or argument - in package pp is parsed and added to the section.
func ensureNamedType(t *doc.Type, sec *section, pp *parsedPackage) {
	typePath := pp.Path + "." + t.Name
	if _, have := sec.Typeset[typePath]; have {
		return
	}

	tt := &namedType{
		Name: t.Name,
		Text: strings.TrimSpace(t.Doc),
	}
	// add it early, so self-referencing types can't cause a loop
	sec.Types = append(sec.Types, tt)
	sec.Typeset[typePath] = struct{}{}

	ts := t.Decl.Specs[0].(*ast.TypeSpec)
	switch nt := ts.Type.(type) {
	case *ast.StructType:
		tt.Kind = typeStruct
		for _, f := range nt.Fields.List {
			ff := &field{
				"",
				nil,
				fieldDoc(f),
				[]*field{},
			}
			ff.Typewords = gatherFieldType(t.Name, ff, f.Type, f.Tag, sec, pp)
			for _, name := range nameList(f.Names, f.Tag) {
				nf := &field{}
				*nf = *ff
				nf.Name = name
				tt.Fields = append(tt.Fields, nf)
			}
		}
	case *ast.Ident:
		if strings.HasSuffix(typePath, "sherpa.Int64s") || strings.HasSuffix(typePath, "sherpa.Uint64s") {
			return
		}

		tt.Text = t.Doc + ts.Comment.Text()
		switch nt.Name {
		case "byte", "int16", "uint16", "int32", "uint32", "int", "uint":
			tt.Kind = typeInts
		case "string":
			tt.Kind = typeStrings
		default:
			log.Fatalf("unrecognized type identifier %#v\n", nt.Name)
		}

		for _, c := range t.Consts {
			for _, spec := range c.Decl.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					log.Fatalf("unsupported non-ast.ValueSpec constant %#v\n", spec)
				}
				if len(vs.Names) != 1 {
					log.Fatalf("unsupported multiple .Names in %#v\n", vs)
				}
				name := vs.Names[0].Name
				if len(vs.Values) != 1 {
					log.Fatalf("unsupported multiple .Values in %#v\n", vs)
				}
				lit, ok := vs.Values[0].(*ast.BasicLit)
				if !ok {
					log.Fatalf("unsupported non-ast.BasicLit first .Values %#v\n", vs)
				}

				comment := vs.Doc.Text() + vs.Comment.Text()
				switch lit.Kind {
				case token.INT:
					if tt.Kind != typeInts {
						log.Fatalf("int value for for non-int-enum %s\n", t.Name)
					}
					v, err := strconv.ParseInt(lit.Value, 10, 64)
					check(err, "parse int literal")
					iv := struct {
						Name  string
						Value int
						Docs  string
					}{name, int(v), comment}
					tt.IntValues = append(tt.IntValues, iv)
				case token.STRING:
					if tt.Kind != typeStrings {
						log.Fatalf("string for non-string-enum %s\n", t.Name)
					}
					v, err := strconv.Unquote(lit.Value)
					check(err, "unquote literal")
					sv := struct {
						Name  string
						Value string
						Docs  string
					}{name, v, comment}
					tt.StringValues = append(tt.StringValues, sv)
				default:
					log.Fatalf("unexpected literal kind %#v\n", lit.Kind)
				}
			}
		}
	default:
		log.Fatalf("unsupported field/param/return type %T\n", ts.Type)
	}
}

// isCommaString returns whether the tag (may be nil) contains a "json:,string" directive.
func isCommaString(tag *ast.BasicLit) bool {
	if tag == nil {
		return false
	}
	st := reflect.StructTag(parseStringLiteral(tag.Value))
	s, ok := st.Lookup("json")
	if !ok || s == "-" {
		return false
	}
	t := strings.Split(s, ",")
	for _, e := range t[1:] {
		if e == "string" {
			return true
		}
	}
	return false
}

func gatherFieldType(typeName string, f *field, e ast.Expr, fieldTag *ast.BasicLit, sec *section, pp *parsedPackage) typewords {
	name := checkReplacedType(e, sec, pp)
	if name != nil {
		return name
	}

	switch t := e.(type) {
	case *ast.Ident:
		tt := pp.lookupType(t.Name)
		if tt != nil {
			ensureNamedType(tt, sec, pp)
			return []string{t.Name}
		}
		commaString := isCommaString(fieldTag)
		name := t.Name
		switch name {
		case "byte":
			name = "uint8"
		case "bool", "int8", "uint8", "int16", "uint16", "int32", "uint32", "float32", "float64", "string":
		case "int64", "uint64":
			if commaString {
				name += "s"
			}
		case "int", "uint":
			name += "32"
		default:
			log.Fatalf("unsupported type %q\n", name)
		}
		if commaString && name != "int64s" && name != "uint64s" {
			log.Fatalf("unsupported tag `json:,\"string\"` for non-64bit int in %s.%s\n", typeName, f.Name)
		}
		return []string{name}
	case *ast.ArrayType:
		return append([]string{"[]"}, gatherFieldType(typeName, f, t.Elt, nil, sec, pp)...)
	case *ast.MapType:
		_ = gatherFieldType(typeName, f, t.Key, nil, sec, pp)
		vt := gatherFieldType(typeName, f, t.Value, nil, sec, pp)
		return append([]string{"{}"}, vt...)
	case *ast.InterfaceType:
		// If we export an interface as an "any" type, we want to make sure it's intended.
		// Require the user to be explicit with an empty interface.
		if t.Methods != nil && len(t.Methods.List) > 0 {
			log.Fatalf("unsupported non-empty interface param/return type %T\n", t)
		}
		return []string{"any"}
	case *ast.StarExpr:
		return append([]string{"nullable"}, gatherFieldType(typeName, f, t.X, fieldTag, sec, pp)...)
	case *ast.SelectorExpr:
		return []string{parseSelector(t, typeName, sec, pp)}
	}
	log.Fatalf("unimplemented ast.Expr %#v for struct %q field %q in gatherFieldType\n", e, typeName, f.Name)
	return nil
}

func parseArgType(e ast.Expr, sec *section, pp *parsedPackage) typewords {
	name := checkReplacedType(e, sec, pp)
	if name != nil {
		return name
	}

	switch t := e.(type) {
	case *ast.Ident:
		tt := pp.lookupType(t.Name)
		if tt != nil {
			ensureNamedType(tt, sec, pp)
			return []string{t.Name}
		}
		name := t.Name
		switch name {
		case "byte":
			name = "uint8"
		case "bool", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64", "string":
		case "int", "uint":
			name += "32"
		case "error":
			// allowed here, checked if in right location by caller
		default:
			log.Fatalf("unsupported type %q\n", name)
		}
		return []string{name}
	case *ast.ArrayType:
		return append([]string{"[]"}, parseArgType(t.Elt, sec, pp)...)
	case *ast.Ellipsis:
		// Ellipsis parameters to a function must be passed as an array, so document it that way.
		return append([]string{"[]"}, parseArgType(t.Elt, sec, pp)...)
	case *ast.MapType:
		_ = parseArgType(t.Key, sec, pp)
		vt := parseArgType(t.Value, sec, pp)
		return append([]string{"{}"}, vt...)
	case *ast.InterfaceType:
		// If we export an interface as an "any" type, we want to make sure it's intended.
		// Require the user to be explicit with an empty interface.
		if t.Methods != nil && len(t.Methods.List) > 0 {
			log.Fatalf("unsupported non-empty interface param/return type %T\n", t)
		}
		return []string{"any"}
	case *ast.StarExpr:
		return append([]string{"nullable"}, parseArgType(t.X, sec, pp)...)
	case *ast.SelectorExpr:
		return []string{parseSelector(t, sec.TypeName, sec, pp)}
	}
	log.Fatalf("unimplemented ast.Expr %#v in parseArgType\n", e)
	return nil
}

func parseSelector(t *ast.SelectorExpr, sourceTypeName string, sec *section, pp *parsedPackage) string {
	packageIdent, ok := t.X.(*ast.Ident)
	if !ok {
		log.Fatalln("unexpected non-ident for SelectorExpr.X")
	}
	pkgName := packageIdent.Name
	typeName := t.Sel.Name

	if pkgName == "time" && typeName == "Time" {
		return "timestamp"
	}
	if pkgName == "sherpa" {
		switch typeName {
		case "Int64s":
			return "int64s"
		case "Uint64s":
			return "uint64s"
		}
	}

	importPath := pp.lookupPackageImportPath(sourceTypeName, pkgName)
	if importPath == "" {
		log.Fatalf("cannot find source for %q (perhaps try -replace)\n", fmt.Sprintf("%s.%s", pkgName, typeName))
	}

	opp := pp.ensurePackageParsed(importPath)
	tt := opp.lookupType(typeName)
	if tt == nil {
		log.Fatalf("could not find type %q in package %q\n", typeName, importPath)
	}
	ensureNamedType(tt, sec, opp)
	return typeName
}

type replacement struct {
	original string // a Go type, eg "pkg.Type" or "*pkg.Type"
	target   typewords
}

var _replacements []replacement

func typeReplacements() []replacement {
	if _replacements != nil {
		return _replacements
	}

	_replacements = []replacement{}
	for _, repl := range strings.Split(*replace, ",") {
		if repl == "" {
			continue
		}
		tokens := strings.Split(repl, " ")
		if len(tokens) < 2 {
			log.Fatalf("bad replacement %q, must have at least two tokens, space-separated\n", repl)
		}
		r := replacement{tokens[0], tokens[1:]}
		_replacements = append(_replacements, r)
	}
	return _replacements
}

// Return a go type name, eg "*time.Time".
// This function does not parse the types itself, because it would mean they could be added to the sherpadoc output even if they aren't otherwise used (due to replacement).
func goTypeName(e ast.Expr, sec *section, pp *parsedPackage) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + goTypeName(t.Elt, sec, pp)
	case *ast.Ellipsis:
		// Ellipsis parameters to a function must be passed as an array, so document it that way.
		return "[]" + goTypeName(t.Elt, sec, pp)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", goTypeName(t.Key, sec, pp), goTypeName(t.Value, sec, pp))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StarExpr:
		return "*" + goTypeName(t.X, sec, pp)
	case *ast.SelectorExpr:
		packageIdent, ok := t.X.(*ast.Ident)
		if !ok {
			log.Fatalln("unexpected non-ident for SelectorExpr.X")
		}
		pkgName := packageIdent.Name
		typeName := t.Sel.Name

		importPath := pp.lookupPackageImportPath(sec.Name, pkgName)
		if importPath != "" {
			return fmt.Sprintf("%s.%s", importPath, typeName)
		}
		return fmt.Sprintf("%s.%s", pkgName, typeName)
		// todo: give proper error message for *ast.StructType
	}
	log.Fatalf("unimplemented ast.Expr %#v in goTypeName\n", e)
	return ""
}

func checkReplacedType(e ast.Expr, sec *section, pp *parsedPackage) typewords {
	repls := typeReplacements()
	if len(repls) == 0 {
		return nil
	}

	name := goTypeName(e, sec, pp)
	return replacementType(repls, name)
}

func replacementType(repls []replacement, name string) typewords {
	for _, repl := range repls {
		if repl.original == name {
			return repl.target
		}
	}
	return nil
}

// Ensures the package for importPath has been parsed at least once, and return it.
func (pp *parsedPackage) ensurePackageParsed(importPath string) *parsedPackage {
	r := pp.Imports[importPath]
	if r != nil {
		return r
	}

	// todo: should also attempt to look at vendor/ directory, and modules
	localPath := os.Getenv("GOPATH")
	if localPath == "" {
		localPath = defaultGOPATH()
	}
	localPath += "/src/" + importPath

	fset := token.NewFileSet()
	pkgs, firstErr := parser.ParseDir(fset, localPath, nil, parser.ParseComments)
	check(firstErr, "parsing code")
	if len(pkgs) != 1 {
		log.Fatalf("need exactly one package parsed for import path %q, but saw %d\n", importPath, len(pkgs))
	}
	for _, pkg := range pkgs {
		docpkg := doc.New(pkg, "", doc.AllDecls)
		npp := &parsedPackage{
			Path:    localPath,
			Pkg:     pkg,
			Docpkg:  docpkg,
			Imports: make(map[string]*parsedPackage),
		}
		pp.Imports[importPath] = npp
		return npp
	}
	return nil
}

// LookupPackageImportPath returns the import/package path for pkgName as used as a selector in this section.
func (pp *parsedPackage) lookupPackageImportPath(sectionTypeName, pkgName string) string {
	file := pp.lookupTypeFile(sectionTypeName)
	for _, imp := range file.Imports {
		if imp.Name != nil && imp.Name.Name == pkgName || imp.Name == nil && strings.HasSuffix(parseStringLiteral(imp.Path.Value), "/"+pkgName) {
			return parseStringLiteral(imp.Path.Value)
		}
	}
	return ""
}

// LookupTypeFile returns the go source file that containst he definition of the type named typeName.
func (pp *parsedPackage) lookupTypeFile(typeName string) *ast.File {
	for _, file := range pp.Pkg.Files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case (*ast.GenDecl):
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.Name == typeName {
							return file
						}
					}
				}
			}
		}
	}
	log.Fatalf("could not find type named %q in package %q\n", typeName, pp.Path)
	return nil
}

// Populate "params" with the arguments from "fields", which are function parameters or return type.
func parseArgs(params *[]sherpadoc.Arg, fields *ast.FieldList, sec *section, pp *parsedPackage, isParams bool) {
	if fields == nil {
		return
	}
	addParam := func(name string, tw typewords) {
		param := sherpadoc.Arg{Name: name, Typewords: tw}
		*params = append(*params, param)
	}
	for _, f := range fields.List {
		typ := parseArgType(f.Type, sec, pp)
		// Handle named params. Can be both arguments to a function or return types.
		for _, name := range f.Names {
			addParam(name.Name, typ)
		}
		// Return types often don't have a name, don't forget them.
		if len(f.Names) == 0 {
			addParam("", typ)
		}
	}

	for i, p := range *params {
		if p.Typewords[len(p.Typewords)-1] != "error" {
			continue
		}
		if isParams || i != len(*params)-1 {
			log.Fatalf("can only have error type as last return value\n")
		}
		pp := *params
		*params = pp[:len(pp)-1]
	}
}

func lowerFirst(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

// ParseMethod ensures the function fn from package pp ends up in section sec, with parameters/return named types filled in.
func parseMethod(sec *section, fn *doc.Func, pp *parsedPackage) {
	f := &function{
		Name:    lowerFirst(fn.Name),
		Text:    fn.Doc,
		Params:  []sherpadoc.Arg{},
		Returns: []sherpadoc.Arg{},
	}

	// If first function parameter is context.Context, we skip it in the documentation.
	// The sherpa handler automatically fills it with the http request context when called.
	params := fn.Decl.Type.Params
	if params != nil && len(params.List) > 0 && len(params.List[0].Names) == 1 && goTypeName(params.List[0].Type, sec, pp) == "context.Context" {
		params.List = params.List[1:]
	}
	isParams := true
	parseArgs(&f.Params, params, sec, pp, isParams)

	isParams = false
	parseArgs(&f.Returns, fn.Decl.Type.Results, sec, pp, isParams)
	sec.Functions = append(sec.Functions, f)
}
