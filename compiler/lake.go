package compiler

import (
	"errors"
	"runtime"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/proc"
)

var Parallelism = runtime.GOMAXPROCS(0) //XXX

func CompileForLake(pctx *proc.Context, program ast.Proc, lake proc.DataAdaptor, parallelism int, head *lakeparse.Commitish) (*Runtime, error) {
	runtime, err := New(pctx, program, lake, head)
	if err != nil {
		return nil, err
	}
	if len(runtime.readers) != 0 {
		return nil, errors.New("query must include a 'from' operator")
	}
	if err := runtime.Optimize(); err != nil {
		return nil, err
	}
	if parallelism == 0 {
		parallelism = Parallelism
	}
	if parallelism > 1 {
		runtime.Parallelize(parallelism)
	}
	if err := runtime.Build(); err != nil {
		return nil, err
	}
	return runtime, nil
}
