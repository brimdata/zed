package compiler

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
	zedruntime "github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/op"
)

var Parallelism = runtime.GOMAXPROCS(0) //XXX

type lakeCompiler struct {
	anyCompiler
	src *data.Source
}

func NewLakeCompiler(r *lake.Root) zedruntime.Compiler {
	// We configure a remote storage engine into the lake compiler so that
	// "from" operators that source http or s3 will work, but stdio and
	// file system accesses will be rejected at open time.
	return &lakeCompiler{src: data.NewSource(storage.NewRemoteEngine(), r)}
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
	return zedruntime.NewQuery(job.pctx, job.Puller(), job.builder.Meter()), nil
}

func (l *lakeCompiler) NewLakeDeleteQuery(pctx *op.Context, program ast.Op, head *lakeparse.Commitish) (*zedruntime.DeleteQuery, error) {
	job, err := newDeleteJob(pctx, program, l.src, head)
	if err != nil {
		return nil, err
	}
	if err := job.Optimize(); err != nil {
		return nil, err
	}
	if err := job.Parallelize(Parallelism); err != nil {
		return nil, err
	}
	if err := job.Build(); err != nil {
		return nil, err
	}
	return zedruntime.NewDeleteQuery(pctx, job.Puller(), job.builder.Deletes()), nil
}

type InvalidDeleteWhereQuery struct{}

func (InvalidDeleteWhereQuery) Error() string {
	return "invalid delete where query: must be a single filter operation"
}

func newDeleteJob(pctx *op.Context, inAST ast.Op, src *data.Source, head *lakeparse.Commitish) (*Job, error) {
	parserAST := ast.Copy(inAST)
	seq, ok := parserAST.(*ast.Sequential)
	if !ok {
		return nil, fmt.Errorf("internal error: AST must begin with a Sequential op: %T", parserAST)
	}
	if len(seq.Ops) == 0 {
		return nil, errors.New("internal error: AST Sequential op cannot be empty")
	}
	if len(seq.Ops) != 1 {
		return nil, &InvalidDeleteWhereQuery{}
	}
	// add trunk
	seq.Prepend(&ast.From{
		Kind: "from",
		Trunks: []ast.Trunk{{
			Kind: "Trunk",
			Source: &ast.Pool{
				Kind:   "Pool",
				Delete: true,
				Spec: ast.PoolSpec{
					Pool: &ast.String{
						Kind: "String",
						Text: "HEAD",
					},
				},
			},
		}},
	})
	entry, err := semantic.Analyze(pctx.Context, seq, src, head)
	if err != nil {
		return nil, err
	}
	if _, ok := entry.Ops[1].(*dag.Filter); !ok {
		return nil, &InvalidDeleteWhereQuery{}
	}
	return &Job{
		pctx:      pctx,
		builder:   kernel.NewBuilder(pctx, src),
		optimizer: optimizer.New(pctx.Context, entry, src),
	}, nil
}
