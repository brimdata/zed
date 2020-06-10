package main

import "C"

import (
	"context"
	"errors"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
)

// result converts an error into response structure expected
// by the Python calling code. cgo does not support exporting
// a function that returns a struct, hence the multiple return
// values.
// If C.CString is used to allocate a C char* string, the Python
// side code will free it.
func result(err error) (*C.char, bool) {
	if err != nil {
		return C.CString(err.Error()), false
	}
	return nil, true
}

// ErrorTest is only used to verify that errors are successfully passed
// between the Go & Python realms.
//
//export ErrorTest
func ErrorTest() (*C.char, bool) {
	return result(errors.New("error test"))
}

//export ZqlFileEval
func ZqlFileEval(inquery, inpath, outpath string) (*C.char, bool) {
	return result(doZqlFileEval(inquery, inpath, outpath))
}

func doZqlFileEval(inquery, inpath, outpath string) error {
	if inpath == "-" {
		inpath = "/dev/stdin"
	}
	if outpath == "-" {
		outpath = "/dev/stdout"
	}
	query, err := zql.ParseProc(inquery)
	if err != nil {
		return err
	}

	zctx := resolver.NewContext()
	rc, err := detector.OpenFile(zctx, inpath, detector.OpenConfig{})
	if err != nil {
		return err
	}
	defer rc.Close()

	w, err := emitter.NewFile(outpath, &zio.WriterFlags{
		Format: "zng",
	})
	if err != nil {
		return err
	}
	defer w.Close()

	fg, err := driver.Compile(context.Background(), query, rc, false, nano.MaxSpan, nil)
	d := driver.NewCLI(w)

	return driver.Run(fg, d, nil)
}

func main() {}
