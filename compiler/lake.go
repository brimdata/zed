package compiler

import (
	"errors"
	goruntime "runtime"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/op"
)

var Parallelism = goruntime.GOMAXPROCS(0) //XXX

type lakeCompiler struct {
	anyCompiler
	lake *lake.Root
}

func NewLakeCompiler(r *lake.Root) runtime.Compiler {
	return &lakeCompiler{lake: r}
}

func (l *lakeCompiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*runtime.Query, error) {
	job, err := NewJob(pctx, program, l.lake, head)
	if err != nil {
		return nil, err
	}
	if len(job.readers) != 0 {
		return nil, errors.New("query must include a 'from' operator")
	}
	if err := job.Optimize(); err != nil {
		return nil, err
	}
	if parallelism == 0 {
		parallelism = Parallelism
	}
	if parallelism > 1 {
		job.Parallelize(parallelism)
	}
	if err := job.Build(); err != nil {
		return nil, err
	}
	return runtime.NewQuery(job.pctx, job.Puller(), job.builder.Meters()), nil
}
