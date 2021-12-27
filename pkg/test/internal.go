package test

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/emitter"
)

type Internal struct {
	Name         string
	Query        string
	Input        string
	InputFormat  string // defaults to "auto", like zq
	OutputFormat string // defaults to "zson", like zq
	Expected     string
	ExpectedErr  error
}

func Trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func stringReader(input string, ifmt string, zctx *zed.Context) (zio.Reader, error) {
	opts := anyio.ReaderOpts{
		Format: ifmt,
	}
	rc := io.NopCloser(strings.NewReader(input))
	return anyio.NewFile(zctx, rc, "test", opts)
}

func newEmitter(ofmt string) (*emitter.Bytes, error) {
	if ofmt == "" {
		ofmt = "zson"
	}
	// XXX text format options not supported
	return emitter.NewBytes(anyio.WriterOpts{Format: ofmt})
}

func (i *Internal) Run() (string, error) {
	program, err := compiler.ParseProc(i.Query)
	if err != nil {
		return "", fmt.Errorf("parse error: %w (%s)", err, i.Query)
	}
	zctx := zed.NewContext()
	reader, err := stringReader(i.Input, i.InputFormat, zctx)
	if err != nil {
		return "", err
	}
	output, err := newEmitter(i.OutputFormat)
	if err != nil {
		return "", err
	}
	q, err := runtime.NewQueryOnReader(context.Background(), zctx, program, reader, nil)
	if err != nil {
		return "", err
	}
	if err := zio.Copy(output, q.AsReader()); err != nil {
		return "", err
	}
	return string(output.Bytes()), nil
}
