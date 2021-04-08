package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/emitter"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/detector"
	"github.com/brimdata/zed/zson"
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

func stringReader(input string, ifmt string, zctx *zson.Context) (zbuf.Reader, error) {
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
	zctx := zson.NewContext()
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
