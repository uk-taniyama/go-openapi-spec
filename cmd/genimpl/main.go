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

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"log"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/deepmap/oapi-codegen/pkg/codegen"
	"github.com/getkin/kin-openapi/openapi3"
)

var (
	flagPackageName  string
	flagOutputFile   string
	flagInputFile    string
	flagTemplateFile string
)

func main() {
	flag.StringVar(&flagPackageName, "p", "", "Package name")
	flag.StringVar(&flagTemplateFile, "t", "", "Server template file")
	flag.StringVar(&flagInputFile, "i", "", "OpenAPI 3.0 spec file")
	flag.StringVar(&flagOutputFile, "o", "", "Output ganarated Server source")
	flag.Parse()
	if flagPackageName == "" || flagTemplateFile == "" || flagInputFile == "" || flagOutputFile == "" {
		fmt.Fprintln(os.Stderr, "Usage of genserverimpl:")
		flag.PrintDefaults()
		os.Exit(-1)
	}

	swagger, err := openapi3.NewLoader().LoadFromFile(flagInputFile)
	if err != nil {
		log.Panic("openapi3.LoadFromFile", err)
	}

	def, err := codegen.OperationDefinitions(swagger)
	if err != nil {
		log.Panic("codegen.OperationDefinitions", err)
	}

	codegen.TemplateFunctions["package"] = func() string { return flagPackageName }
	codegen.TemplateFunctions["_"] = func() string { return "" }
	codegen.TemplateFunctions["getStatusText"] = func(code string) string {
		n, err := strconv.Atoi(code)
		if err != nil {
			return ""
		}
		return http.StatusText(n)
	}

	t := template.New("codegen").Funcs(codegen.TemplateFunctions)

	out := bytes.Buffer{}
	data, err := os.ReadFile(flagTemplateFile)
	if err != nil {
		log.Panic("os.ReadFile template", err)
	}
	_, err = t.Parse(string(data))
	if err != nil {
		log.Panic("template.Parse", err)
	}
	err = t.Execute(&out, def)
	if err != nil {
		log.Panic("template.Execute", err)
	}

	// Final gofmt.
	src, err := format.Source(out.Bytes())
	if err != nil {
		log.Panicf("format.Source: %v on\n%s", err, src)
	}

	if err := os.WriteFile(flagOutputFile, src, 0644); err != nil {
		log.Panicf("os.WriteFile: %v\n", err)
	}
}
