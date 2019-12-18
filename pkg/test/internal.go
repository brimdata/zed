package test

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/driver"
	"github.com/mccanne/zq/emitter"
	"github.com/mccanne/zq/pkg/zio/detector"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/zql"
)

type Internal struct {
	Name     string
	Query    string
	Input    string
	Format   string
	Expected string
}

func Trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func stringReader(input string, r *resolver.Table) (zson.Reader, error) {
	return detector.NewReader(strings.NewReader(input), r)
}

func (i *Internal) Run() (string, error) {
	program, err := zql.ParseProc(i.Query)
	if err != nil {
		return "", fmt.Errorf("parse error: %s (%s)", err, i.Query)
	}
	reader, err := stringReader(i.Input, resolver.NewTable())
	if err != nil {
		return "", err
	}
	mux, err := driver.Compile(program, scanner.NewScanner(reader, nil))
	if err != nil {
		return "", err
	}
	// XXX text format options not supported and passed in as nil
	output, err := emitter.NewBytes(i.Format, nil)
	if err != nil {
		return "", err
	}
	runner := driver.New(output)
	if err := runner.Run(mux); err != nil {
		return "", err
	}
	return string(output.Bytes()), nil
}
