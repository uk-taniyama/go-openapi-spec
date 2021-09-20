package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"strings"

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

func main() {
	validate := validator.New()

	config := Config{}
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

	var fset = token.NewFileSet()
	af, err := parser.ParseFile(fset, config.InputFile, nil, 0)
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
}
