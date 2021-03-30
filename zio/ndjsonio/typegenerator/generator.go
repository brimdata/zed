package main

import (
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"

	"github.com/brimdata/zed/cli/inputflags"
)

func main() {
	var outName, packageName, varName string

	flag.StringVar(&outName, "o", "", "output filename")
	flag.StringVar(&packageName, "package", "main", "package name")
	flag.StringVar(&varName, "var", "tc", "variable name")

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	fileName := flag.Arg(0)
	tc, err := inputflags.LoadJSONConfig(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing json config: %s\n", err)
		return
	}

	contents := fmt.Sprintf(`package %s

import "github.com/brimdata/zed/zio/ndjsonio"

var %s *ndjsonio.TypeConfig = %#v`, packageName, varName, tc)

	formatted, err := format.Source([]byte(contents))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error formatting code: %s\n", err)
		return
	}

	err = ioutil.WriteFile(outName, formatted, 0644)
	if err != nil {
		log.Fatalf("Error writing to %s: %s\n", outName, err)
	}
}
