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

	g, err := genspec.New(&config)
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
