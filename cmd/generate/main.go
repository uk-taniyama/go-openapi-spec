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
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/uk-taniyama/go-openapi-spec/pkg/genspec"
)

func main() {
	config := genspec.Config{}

	flag.BoolVar(&config.Debug, "d", false, "debug")
	flag.StringVar(&config.PackageName, "p", "", "PackageName")
	flag.StringVar(&config.InputFile, "i", "", "InputFile OpenAPI spec.go file")
	flag.StringVar(&config.OutputFile, "o", "", "OutputFile ganarated OpenAPI spec")
	flag.Parse()

	g, err := genspec.NewGenerator(&config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Error:\n\t%v\n", strings.Join(strings.Split(err.Error(), "\n"), "\n\t"))
		os.Exit(-1)
	}
	err = g.Run()
	if err != nil {
		log.Panicln(err.Error())
	}
}
