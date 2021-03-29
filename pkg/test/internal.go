package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/brimdata/zq/compiler"
	"github.com/brimdata/zq/driver"
	"github.com/brimdata/zq/emitter"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zio"
	"github.com/brimdata/zq/zio/detector"
	"github.com/brimdata/zq/zng/resolver"
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
	opts := zio.ReaderOpts{
		Format: ifmt,
	}
	rc := ioutil.NopCloser(strings.NewReader(input))

	return detector.OpenFromNamedReadCloser(zctx, rc, "test", opts)
}

func newEmitter(ofmt string) (*emitter.Bytes, error) {
	if ofmt == "" {
		ofmt = "tzng"
	}
	// XXX text format options not supported
	return emitter.NewBytes(zio.WriterOpts{Format: ofmt})
}

func (i *Internal) Run() (string, error) {
	program, err := compiler.ParseProc(i.Query)
	if err != nil {
		return "", fmt.Errorf("parse error: %s (%s)", err, i.Query)
	}
	zctx := resolver.NewContext()
	reader, err := stringReader(i.Input, i.InputFormat, zctx)
	if err != nil {
		return "", err
	}
	output, err := newEmitter(i.OutputFormat)
	if err != nil {
		return "", err
	}
	d := driver.NewCLI(output)
	if err := driver.Run(context.Background(), d, program, zctx, reader, driver.Config{}); err != nil {
		return "", err
	}
	return string(output.Bytes()), nil
}
