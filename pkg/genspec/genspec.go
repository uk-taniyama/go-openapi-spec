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
	"fmt"
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
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
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

	kv := KeyValue{}
	err = Convert(g.spec, &kv)
	if err != nil {
		return errors.Wrap(err, "Convert Scheme to KeyValue")
	}
	// KeyValue -> MapSlice
	ms := yaml.MapSlice{}
	keys := []string{"openapi", "info", "paths", "components"}
	for _, k := range keys {
		v := kv[k]
		if v != nil {
			ms = append(ms, yaml.MapItem{
				Key:   k,
				Value: v,
			})
			delete(kv, k)
		}
	}
	for k, v := range kv {
		ms = append(ms, yaml.MapItem{
			Key:   k,
			Value: v,
		})
	}
	bytes, err := yaml.Marshal(&ms)
	if err != nil {
		return errors.Wrap(err, "yaml.Marshal")
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
			name := strcase.ToLowerCamel(f.Names[0].Name)
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

func getResponse(ope *openapi3.Operation, name string) *openapi3.Response {
	ref := ope.Responses[name]
	if ref == nil {
		ref = &openapi3.ResponseRef{
			Value: &openapi3.Response{},
		}
		ope.Responses[name] = ref
	}
	return ref.Value
}

func (g *Generator) setResponseDesc(ope *openapi3.Operation, name string, desc string) {
	res := getResponse(ope, name)
	res.Description = &desc
}

func (g *Generator) setResponse(ope *openapi3.Operation, name string, ref *openapi3.SchemaRef) {
	res := getResponse(ope, name)
	res.Content = openapi3.Content{
		"application/json": &openapi3.MediaType{
			Schema: ref,
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

var codePattern = regexp.MustCompile("^[1-9][0-9][0-9]$")

func isResCode(name string) bool {
	if codePattern.MatchString(name) {
		return true
	}
	return name == "default"
}

type OpeDoc struct {
	Desc   string
	Method string
	Path   string
	KV     KeyValue
}

var PathPattern = regexp.MustCompile("\\(([A-Z]+) (/.+)\\)")

func ParseOpeDoc(doc string) *OpeDoc {
	lines := strings.Split(doc, "\n")
	for i, l := range lines {
		g := PathPattern.FindStringSubmatch(l)
		if len(g) > 0 {
			desc := strings.Join(lines[:i], "\n")
			rest := strings.Join(lines[i+1:], "\n")
			kv := KeyValue{}
			err := yaml.Unmarshal([]byte(rest), &kv)
			if err != nil {
				log.Panic("ParseOpeDoc:", err)
			}
			return &OpeDoc{
				Desc:   strings.TrimSpace(desc),
				Method: g[1],
				Path:   g[2],
				KV:     kv,
			}
		}
	}
	return nil
}

func (g *Generator) generatePaths(ts *ast.TypeSpec, i *ast.InterfaceType) {
	for _, m := range i.Methods.List {
		name := m.Names[0].Name
		opeDoc := ParseOpeDoc(m.Doc.Text())
		if opeDoc == nil {
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
		g.setOperation(opeDoc.Path, opeDoc.Method, ope)
		if opeDoc.Desc != "" {
			ope.Description = opeDoc.Desc
		}
		for k, v := range opeDoc.KV {
			if isResCode(k) {
				g.setResponseDesc(ope, k, fmt.Sprintf("%v", v))
			}
		}

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

		g.setResponse(ope, "default", &openapi3.SchemaRef{
			Ref: "#/components/schemas/" + "Error",
		})
		if ft.Results != nil && len(ft.Results.List) > 0 {
			for _, r := range ft.Results.List {
				g.appendResponse(ope, r)
			}
		}
	}
}
