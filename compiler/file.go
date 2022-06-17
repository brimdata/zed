package compiler

import (
	"errors"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zio"
)

type fsCompiler struct {
	anyCompiler
	fs op.DataAdaptor
}

func NewFileSystemCompiler(adaptor op.DataAdaptor) runtime.Compiler {
	return &fsCompiler{fs: adaptor}
}

func (f *fsCompiler) NewQuery(pctx *op.Context, o ast.Op, readers []zio.Reader) (*runtime.Query, error) {
	job, err := NewJob(pctx, o, f.fs, nil)
	if err != nil {
		return nil, err
	}
	if isJoin(o) {
		if len(readers) != 2 {
			return nil, errors.New("join operaetor requires two inputs")
		}
		if len(job.readers) != 2 {
			return nil, errors.New("internal error: join expected by semantic analyzer")
		}
		job.readers[0].Readers = readers[0:1]
		job.readers[1].Readers = readers[1:2]
	} else if len(readers) == 0 {
		// If there's no reader but the DAG wants an input, then
		// flag an error.
		if len(job.readers) != 0 {
			return nil, errors.New("no input specified: use a command-line file or a Zed source operator")
		}
	} else {
		// If there's a reader but the DAG doesn't want an input,
		// then flag an error.
		// TBD: we could have such a configuration is a composite
		// from command includes a "pass" operator, but we can add this later.
		// See issue #2640.
		if len(job.readers) == 0 {
			return nil, errors.New("redundant inputs specified: use either command-line files or a Zed source operator")
		}
		if len(job.readers) != 1 {
			return nil, errors.New("Zed query requires a single input path")
		}
		job.readers[0].Readers = readers
	}
	return optimizeAndBuild(job)
}

func (*fsCompiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*runtime.Query, error) {
	panic("NewLakeQuery called on compiler.fsCompiler")
}

func isJoin(o ast.Op) bool {
	seq, ok := o.(*ast.Sequential)
	if !ok || len(seq.Ops) == 0 {
		return false
	}
	_, ok = seq.Ops[0].(*ast.Join)
	return ok
}

func optimizeAndBuild(job *Job) (*runtime.Query, error) {
	// Call optimize to possible push down a filter predicate into the
	// kernel.Reader so that the zng scanner can do boyer-moore.
	if err := job.Optimize(); err != nil {
		return nil, err
	}
	// For an internal reader (like a shaper on intake), we don't do
	// any parallelization right now though this could be potentially
	// beneficial depending on where the bottleneck is for a given shaper.
	// See issue #2641.
	if err := job.Build(); err != nil {
		return nil, err
	}
	return runtime.NewQuery(job.pctx, job.Puller(), job.builder.Meters()), nil
}
