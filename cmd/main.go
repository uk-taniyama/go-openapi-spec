package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/go-playground/validator.v9"
)

var (
	flagPackageName  string
	flagOutputFile   string
	flagInputFile    string
	flagTemplateFile string
)

type Config struct {
	debug       bool
	PackageName string // `validate:"required"`
	InputFile   string `validate:"required"`
	OutputFile  string
}

func getWriter(out string) (io.Writer, error) {
	if out == "" {
		return os.Stdout, nil
	}
	return os.Create(out)
}

func ref(val *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: val}
}

var fset = token.NewFileSet()
var config = Config{}
var spec = openapi3.T{}

func main() {
	validate := validator.New()

	flag.BoolVar(&config.debug, "d", false, "debug")
	flag.StringVar(&config.PackageName, "p", "", "PackageName")
	flag.StringVar(&config.InputFile, "i", "", "InputFile OpenAPI spec.go file")
	flag.StringVar(&config.OutputFile, "o", "", "OutputFile ganarated OpenAPI spec")
	flag.Parse()

	err := validate.Struct(&config)
	if config.debug && config.InputFile != "" {
		err = nil
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Error:\n\t%v\n", strings.Join(strings.Split(err.Error(), "\n"), "\n\t"))
		os.Exit(-1)
	}

	af, err := parser.ParseFile(fset, config.InputFile, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	w, err := getWriter(config.OutputFile)
	if err != nil {
		log.Fatal(err)
	}

	if config.debug {
		ast.Fprint(w, fset, af, nil)
		return
	}

	generate(af)

	bytes, err := json.Marshal(spec)
	if err != nil {
		log.Fatal(err)
	}
	_, err = w.Write(bytes)
	if err != nil {
		log.Fatal(err)
	}
}

func generate(af *ast.File) {
	spec.OpenAPI = "3.0.0"
	spec.Components.Schemas = openapi3.Schemas{}
	spec.Paths = openapi3.Paths{}
	for _, d := range af.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch i := ts.Type.(type) {
			case *ast.StructType:
				generateComponent(ts, i)
			case *ast.InterfaceType:
				generatePaths(ts, i)
			}
		}
	}
}

func generateComponent(ts *ast.TypeSpec, s *ast.StructType) {
	if ts.Name == nil {
		log.Panicf("%+v", ts)
	}

	schemas := spec.Components.Schemas
	schemas[ts.Name.Name] = fromStruct(s)
}

func fromStruct(s *ast.StructType) *openapi3.SchemaRef {
	parentRef := ""
	required := []string{}
	properties := openapi3.Schemas{}

	for _, f := range s.Fields.List {
		if len(f.Names) == 0 {
			i, ok := f.Type.(*ast.Ident)
			if !ok {
				log.Panic("Cannot get indent")
			}
			if parentRef != "" {
				log.Panic("Cannot get indent")
			}
			parentRef = "#/components/schemas/" + i.Name
		} else {
			name := f.Names[0].Name
			properties[name] = fromType(f.Type)
		}
	}

	schema := &openapi3.Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	if parentRef == "" {
		return ref(schema)
	}
	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			AllOf: openapi3.SchemaRefs{
				{Ref: parentRef},
				{Value: schema},
			},
		},
	}
}

func fromType(expr ast.Expr) *openapi3.SchemaRef {
	ref := &openapi3.SchemaRef{}

	switch i := expr.(type) {
	case *ast.Ident:
		return fromIdent(i)
	case *ast.ArrayType:
		return fromArrayType(i)
	default:
		log.Println("  ", "? ")
	}

	return ref
}

func fromArrayType(i *ast.ArrayType) *openapi3.SchemaRef {
	return ref(
		&openapi3.Schema{
			Type:  "array",
			Items: fromType(i.Elt),
		},
	)
}

func fromIdent(i *ast.Ident) *openapi3.SchemaRef {
	switch i.Name {
	case "int", "int32", "int64":
		return ref(&openapi3.Schema{Type: "integer"})
	case "string":
		return ref(&openapi3.Schema{Type: "string"})
	default:
		return &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + i.Name,
		}
	}
}

var path = regexp.MustCompile("\\(([A-Z]+) (/.+)\\)")

func lookupSchema(expr ast.Expr) *openapi3.SchemaRef {
	i, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}
	return spec.Components.Schemas[i.Name]
}

func appendQuery(ope *openapi3.Operation, expr ast.Expr) {
	ref := lookupSchema(expr)
	if ref == nil || ref.Value == nil || ref.Value.Type != "object" {
		return
	}
	for n, p := range ref.Value.Properties {
		ope.Parameters = append(ope.Parameters, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:   n,
				In:     "query",
				Schema: p,
			},
		})
	}
}

func appendBody(ope *openapi3.Operation, expr ast.Expr) {
	ref := fromType(expr)
	ope.RequestBody = &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Required: true,
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: ref,
				},
			},
		},
	}
}

func appendPath(ope *openapi3.Operation, name string, expr ast.Expr) {
	ope.Parameters = append(ope.Parameters, &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:     name,
			In:       "path",
			Required: true,
			Schema:   fromType(expr),
		},
	})
}

func setResponse(ope *openapi3.Operation, name string, ref *openapi3.SchemaRef) {
	ope.Responses[name] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: ref,
				},
			},
		},
	}
}

func appendResponse(ope *openapi3.Operation, r *ast.Field) {
	setResponse(ope, "200", fromType(r.Type))
}

func setOperation(path string, method string, ope *openapi3.Operation) {
	p := spec.Paths[path]
	if p == nil {
		p = &openapi3.PathItem{}
		spec.Paths[path] = p
	}
	p.SetOperation(method, ope)
}

func generatePaths(ts *ast.TypeSpec, i *ast.InterfaceType) {
	for _, m := range i.Methods.List {
		group := path.FindStringSubmatch(m.Doc.Text())
		if len(group) == 0 {
			log.Panic("TODO 見つからん( ﾉД`)ｼｸｼｸ…")
		}
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			log.Panic("TODO FuncTypeじゃないじゃん( ﾉД`)ｼｸｼｸ…")
		}

		ope := &openapi3.Operation{
			OperationID: m.Names[0].Name,
			Parameters:  openapi3.Parameters{},
			Responses:   openapi3.Responses{},
		}
		method := strings.ToUpper(group[1])
		path := group[2]
		setOperation(path, method, ope)
		setResponse(ope, "default", &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + "Error",
		})
		for _, p := range ft.Params.List {
			name := p.Names[0].Name
			switch name {
			case "params":
				appendQuery(ope, p.Type)
			case "body":
				appendBody(ope, p.Type)
			default:
				appendPath(ope, name, p.Type)
			}
		}

		if ft.Results != nil && len(ft.Results.List) > 0 {
			for _, r := range ft.Results.List {
				appendResponse(ope, r)
			}
		}
	}
}
