package compiler

import (
	"errors"
	goruntime "runtime"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec"
)

var Parallelism = goruntime.GOMAXPROCS(0) //XXX

type lakeCompiler struct {
	anyCompiler
	src *data.Source
}

func NewLakeCompiler(r *lake.Root) runtime.Compiler {
	// We configure a remote storage engine into the lake compiler so that
	// "from" operators that source http or s3 will work, but stdio and
	// file system accesses will be rejected at open time.
	return &lakeCompiler{src: data.NewSource(storage.NewRemoteEngine(), r)}
}

func (l *lakeCompiler) NewLakeQuery(rctx *runtime.Context, program ast.Seq, parallelism int, head *lakeparse.Commitish) (runtime.Query, error) {
	job, err := NewJob(rctx, program, l.src, head)
	if err != nil {
		return nil, err
	}
	if _, ok := job.DefaultScan(); ok {
		return nil, errors.New("query must include a 'from' operator")
	}
	if err := job.Optimize(); err != nil {
		return nil, err
	}
	if parallelism == 0 {
		parallelism = Parallelism
	}
	if parallelism > 1 {
		if err := job.Parallelize(parallelism); err != nil {
			return nil, err
		}
	}
	if err := job.Build(); err != nil {
		return nil, err
	}
	return exec.NewQuery(job.rctx, job.Puller(), job.builder.Meter()), nil
}

func (l *lakeCompiler) NewLakeDeleteQuery(rctx *runtime.Context, program ast.Seq, head *lakeparse.Commitish) (runtime.DeleteQuery, error) {
	job, err := newDeleteJob(rctx, program, l.src, head)
	if err != nil {
		return nil, err
	}
	if err := job.OptimizeDeleter(Parallelism); err != nil {
		return nil, err
	}
	if err := job.Build(); err != nil {
		return nil, err
	}
	return exec.NewDeleteQuery(rctx, job.Puller(), job.builder.Deletes()), nil
}

type InvalidDeleteWhereQuery struct{}

func (InvalidDeleteWhereQuery) Error() string {
	return "invalid delete where query: must be a single filter operation"
}

func newDeleteJob(rctx *runtime.Context, in ast.Seq, src *data.Source, head *lakeparse.Commitish) (*Job, error) {
	seq := ast.CopySeq(in)
	if len(seq) == 0 {
		return nil, errors.New("internal error: AST seq cannot be empty")
	}
	if len(seq) != 1 {
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
	entry, err := semantic.Analyze(rctx.Context, seq, src, head)
	if err != nil {
		return nil, err
	}
	if _, ok := entry[1].(*dag.Filter); !ok {
		return nil, &InvalidDeleteWhereQuery{}
	}
	return &Job{
		rctx:      rctx,
		builder:   kernel.NewBuilder(rctx, src),
		optimizer: optimizer.New(rctx.Context, src),
		entry:     entry,
	}, nil
}
