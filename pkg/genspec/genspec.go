// Copyright (c) 2021 uk-taniyama.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genspec

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
)

var (
	flagPackageName  string
	flagOutputFile   string
	flagInputFile    string
	flagTemplateFile string
)

type Config struct {
	Debug       bool
	PackageName string // `validate:"required"`
	InputFile   string `validate:"required"`
	OutputFile  string
}

func getWriter(out string) (io.Writer, error) {
	if out == "" {
		return os.Stdout, nil
	}
	w, err := os.Create(out)
	if err != nil {
		return nil, errors.Wrap(err, "os.Create")
	}
	return w, nil
}

func ref(val *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: val}
}

type Generator struct {
	config *Config
	fset   *token.FileSet
	spec   *openapi3.T
}

func New(config *Config) (*Generator, error) {
	if config.Debug && config.InputFile != "" {
	} else {
		validate := validator.New()
		err := validate.Struct(config)
		if err != nil {
			return nil, err
		}
	}
	return &Generator{
		config: config,
		fset:   token.NewFileSet(),
		spec:   &openapi3.T{},
	}, nil
}

func (g *Generator) Run() error {
	af, err := parser.ParseFile(g.fset, g.config.InputFile, nil, parser.ParseComments)
	if err != nil {
		return errors.Wrap(err, "parser.ParseFile")
	}
	w, err := getWriter(g.config.OutputFile)
	if err != nil {
		return err
	}

	if g.config.Debug {
		ast.Fprint(w, g.fset, af, nil)
		return nil
	}

	g.generate(af)

	bytes, err := json.Marshal(g.spec)
	if err != nil {
		return errors.Wrap(err, "json.Marshal")
	}
	bytes, err = yaml.JSONToYAML(bytes)
	if err != nil {
		return errors.Wrap(err, "yaml.JSONToYAML")
	}
	_, err = w.Write(bytes)
	if err != nil {
		return errors.Wrap(err, "w.Write")
	}
	return nil
}

func (g *Generator) generate(af *ast.File) {
	g.spec.OpenAPI = "3.0.0"
	g.spec.Components.Schemas = openapi3.Schemas{}
	g.spec.Paths = openapi3.Paths{}

	for _, d := range af.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range gd.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if ok {
				g.importSpec(vs)
			}
			ts, ok := s.(*ast.TypeSpec)
			if ok {
				switch i := ts.Type.(type) {
				case *ast.StructType:
					g.generateComponent(ts, i)
				case *ast.InterfaceType:
					g.generatePaths(ts, i)
				}
			}
		}
	}
}

func (g *Generator) importSpec(vs *ast.ValueSpec) {
	if vs.Names == nil || vs.Names[0].Name != "OpenAPISpec" {
		return
	}
	if vs.Values == nil || len(vs.Values) != 1 {
		log.Panic("Invalid OpenAPISpec", vs.Values)
	}
	bl, ok := vs.Values[0].(*ast.BasicLit)
	if !ok {
		log.Panic("Invalid OpenAPISpec", vs.Values)
	}
	spec, err := strconv.Unquote(bl.Value)
	if err != nil {
		log.Panic("Invalid OpenAPISpec", vs.Values)
	}
	t, err := openapi3.NewLoader().LoadFromData([]byte(spec))
	if err != nil {
		log.Panic("Invalid OpenAPISpec", spec)
	}

	t.OpenAPI = g.spec.OpenAPI
	t.Components = g.spec.Components
	t.Paths = g.spec.Paths

	g.spec = t
}

func (g *Generator) generateComponent(ts *ast.TypeSpec, s *ast.StructType) {
	if ts.Name == nil {
		log.Panicf("%+v", ts)
	}

	schemas := g.spec.Components.Schemas
	schemas[ts.Name.Name] = g.fromStruct(s)
}

func (g *Generator) fromStruct(s *ast.StructType) *openapi3.SchemaRef {
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
				log.Panic("Already exist parent")
			}
			parentRef = "#/components/schemas/" + i.Name
		} else {
			name := f.Names[0].Name
			prop := fromType(f.Type)
			required = append(required, name)
			if f.Tag != nil {
				tag, err := strconv.Unquote(f.Tag.Value)
				if err == nil {
					setSchemaFromTag(prop, tag)
				}
			}
			properties[name] = prop
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

	return ref(&openapi3.Schema{
		AllOf: openapi3.SchemaRefs{
			{Ref: parentRef},
			{Value: schema},
		},
	})
}

func setSchemaFromTag(ref *openapi3.SchemaRef, tag string) {
	if ref == nil || ref.Value == nil {
		return
	}
	kv := ParseTag(tag)
	if len(kv) == 0 {
		return
	}
	ExpandTagForScheme(ref.Value, kv)
}

func fromType(expr ast.Expr) *openapi3.SchemaRef {
	switch i := expr.(type) {
	case *ast.Ident:
		return fromIdent(i)
	case *ast.ArrayType:
		return fromArrayType(i)
	default:
		log.Panicf("Unkown type.: %v", expr)
		return nil
	}
}

func fromArrayType(i *ast.ArrayType) *openapi3.SchemaRef {
	return ref(&openapi3.Schema{
		Type:  "array",
		Items: fromType(i.Elt),
	})
}

var DefaultIdentScheme = map[string]*openapi3.Schema{
	"int":    {Type: "integer"},
	"int32":  {Type: "integer", Format: "int32"},
	"int64":  {Type: "integer", Format: "int64"},
	"string": {Type: "string"},
}

func fromIdent(i *ast.Ident) *openapi3.SchemaRef {
	s := DefaultIdentScheme[i.Name]
	if s != nil {
		return ref(s)
	}

	return &openapi3.SchemaRef{
		Ref: "#/components/schemas/" + i.Name,
	}
}

func (g *Generator) lookupSchema(expr ast.Expr) *openapi3.SchemaRef {
	i, ok := expr.(*ast.Ident)
	if !ok {
		return nil
	}
	return g.spec.Components.Schemas[i.Name]
}

func (g *Generator) appendQuery(ope *openapi3.Operation, expr ast.Expr) {
	ref := g.lookupSchema(expr)
	if ref == nil || ref.Value == nil || ref.Value.Type != "object" {
		log.Panicf("Unexpected expr: %v", ref)
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

func (g *Generator) appendBody(ope *openapi3.Operation, expr ast.Expr) {
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

func (g *Generator) appendPath(ope *openapi3.Operation, name string, expr ast.Expr) {
	ope.Parameters = append(ope.Parameters, &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:     name,
			In:       "path",
			Required: true,
			Schema:   fromType(expr),
		},
	})
}

func (g *Generator) setResponse(ope *openapi3.Operation, name string, ref *openapi3.SchemaRef) {
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

func (g *Generator) appendResponse(ope *openapi3.Operation, r *ast.Field) {
	g.setResponse(ope, "200", fromType(r.Type))
}

func (g *Generator) setOperation(path string, method string, ope *openapi3.Operation) {
	p := g.spec.Paths[path]
	if p == nil {
		p = &openapi3.PathItem{}
		g.spec.Paths[path] = p
	}
	p.SetOperation(method, ope)
}

var path = regexp.MustCompile("\\(([A-Z]+) (/.+)\\)")

func (g *Generator) generatePaths(ts *ast.TypeSpec, i *ast.InterfaceType) {
	for _, m := range i.Methods.List {
		name := m.Names[0].Name
		group := path.FindStringSubmatch(m.Doc.Text())
		if len(group) == 0 {
			log.Panicf("Not found pathinfo: %v\n", name)
		}
		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			log.Panicf("Not FuncType: %v\n", name)
		}

		ope := &openapi3.Operation{
			OperationID: name,
			Parameters:  openapi3.Parameters{},
			Responses:   openapi3.Responses{},
		}
		method := strings.ToUpper(group[1])
		path := group[2]
		g.setOperation(path, method, ope)
		g.setResponse(ope, "default", &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + "Error",
		})

		for _, p := range ft.Params.List {
			name := p.Names[0].Name
			switch name {
			case "params":
				g.appendQuery(ope, p.Type)
			case "body":
				g.appendBody(ope, p.Type)
			default:
				g.appendPath(ope, name, p.Type)
			}
		}

		if ft.Results != nil && len(ft.Results.List) > 0 {
			for _, r := range ft.Results.List {
				g.appendResponse(ope, r)
			}
		}
	}
}
