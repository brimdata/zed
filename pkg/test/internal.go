package test

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
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

func stringReader(input string, ifmt string, zctx *resolver.Context) (zbuf.Reader, error) {
	if ifmt == "" {
		return detector.NewReader(strings.NewReader(input), zctx)
	}
	zr, err := detector.LookupReader(ifmt, strings.NewReader(input), zctx)
	if err != nil {
		return nil, err
	}
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
	reader, err := stringReader(i.Input, i.InputFormat, resolver.NewContext())
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
