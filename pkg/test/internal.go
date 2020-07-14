package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
)

type Internal struct {
	Name         string
	Query        string
	Input        string
	InputFormat  string // defaults to "auto", like zq
	OutputFormat string // defaults to "tzng", like zq
	Expected     string
	ExpectedErr  error
}

func Trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func stringReader(input string, ifmt string, zctx *resolver.Context) (zbuf.Reader, error) {
	cfg := detector.OpenConfig{
		Format: ifmt,
	}
	rc := ioutil.NopCloser(strings.NewReader(input))

	return detector.OpenFromNamedReadCloser(zctx, rc, "test", cfg)
}

func newEmitter(ofmt string) (*emitter.Bytes, error) {
	if ofmt == "" {
		ofmt = "tzng"
	}
	// XXX text format options not supported
	return emitter.NewBytes(&zio.WriterFlags{Format: ofmt})
}

func (i *Internal) Run() (string, error) {
	program, err := zql.ParseProc(i.Query)
	if err != nil {
		return "", fmt.Errorf("parse error: %s (%s)", err, i.Query)
	}
	zctx := resolver.NewContext()
	reader, err := stringReader(i.Input, i.InputFormat, zctx)
	if err != nil {
		return "", err
	}
	mux, err := driver.Compile(context.Background(), program, zctx, reader, driver.Config{})
	if err != nil {
		return "", err
	}
	output, err := newEmitter(i.OutputFormat)
	if err != nil {
		return "", err
	}
	d := driver.NewCLI(output)
	if err := driver.Run(mux, d, nil); err != nil {
		return "", err
	}
	return string(output.Bytes()), nil
}
