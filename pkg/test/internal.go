package test

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/driver"
	"github.com/mccanne/zq/emitter"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zio/detector"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/mccanne/zq/zql"
)

type Internal struct {
	Name         string
	Query        string
	Input        string
	InputFormat  string // defaults to "auto", like zq
	OutputFormat string // defaults to "zng", like zq
	Expected     string
	ExpectedErr  error
}

func Trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func stringReader(input string, ifmt string, r *resolver.Table) (zbuf.Reader, error) {
	if ifmt == "" {
		return detector.NewReader(strings.NewReader(input), r)
	}
	zr := detector.LookupReader(ifmt, strings.NewReader(input), r)
	if zr == nil {
		return nil, fmt.Errorf("unknown input format %s", ifmt)
	}
	return zr, nil
}

func newEmitter(ofmt string) (*emitter.Bytes, error) {
	if ofmt == "" {
		ofmt = "zng"
	}
	// XXX text format options not supported and passed in as nil
	return emitter.NewBytes(ofmt, nil)
}

func (i *Internal) Run() (string, error) {
	program, err := zql.ParseProc(i.Query)
	if err != nil {
		return "", fmt.Errorf("parse error: %s (%s)", err, i.Query)
	}
	reader, err := stringReader(i.Input, i.InputFormat, resolver.NewTable())
	if err != nil {
		return "", err
	}
	mux, err := driver.Compile(program, scanner.NewScanner(reader, nil))
	if err != nil {
		return "", err
	}
	output, err := newEmitter(i.OutputFormat)
	if err != nil {
		return "", err
	}
	runner := driver.New(output)
	if err := runner.Run(mux); err != nil {
		return "", err
	}
	return string(output.Bytes()), nil
}
