package compiler

import (
	"errors"
	"runtime"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
	zedruntime "github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
)

var Parallelism = runtime.GOMAXPROCS(0) //XXX

type lakeCompiler struct {
	anyCompiler
	src *querygen.Source
}

func NewLakeCompiler(r *lake.Root) zedruntime.Compiler {
	// We configure a remote storage engine into the lake compiler so that
	// "from" operators that source http or s3 will work, but stdio and
	// file system accesses will be rejected at open time.
	return &lakeCompiler{src: querygen.NewSource(storage.NewRemoteEngine(), r)}
}

func (l *lakeCompiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*zedruntime.Query, error) {
	job, err := NewJob(pctx, program, l.src, head)
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
	return zedruntime.NewQuery(job.pctx, job.Puller(), job.builder.Meters()), nil
}
