package compiler

import (
	"errors"

	"github.com/brimdata/super/compiler/ast"
	"github.com/brimdata/super/compiler/data"
	"github.com/brimdata/super/lakeparse"
	"github.com/brimdata/super/pkg/storage"
	"github.com/brimdata/super/runtime"
	"github.com/brimdata/super/runtime/exec"
	"github.com/brimdata/super/zio"
)

type fsCompiler struct {
	anyCompiler
	src *data.Source
}

func NewFileSystemCompiler(engine storage.Engine) runtime.Compiler {
	return &fsCompiler{src: data.NewSource(engine, nil)}
}

func (f *fsCompiler) NewQuery(rctx *runtime.Context, seq ast.Seq, readers []zio.Reader) (runtime.Query, error) {
	job, err := NewJob(rctx, seq, f.src, nil)
	if err != nil {
		return nil, err
	}
	if len(readers) == 0 {
		// If there's no reader but the DAG wants an input, then
		// flag an error.
		if _, ok := job.DefaultScan(); ok {
			return nil, errors.New("no input specified: use a command-line file or a Zed source operator")
		}
	} else {
		// If there's a reader but the DAG doesn't want an input,
		// then flag an error.
		// TBD: we could have such a configuration is a composite
		// from command includes a "pass" operator, but we can add this later.
		// See issue #2640.
		if _, ok := job.DefaultScan(); !ok {
			return nil, errors.New("redundant inputs specified: use either command-line files or a Zed source operator")
		}
	}
	return optimizeAndBuild(job, readers)
}

func (*fsCompiler) NewLakeQuery(_ *runtime.Context, program ast.Seq, parallelism int, head *lakeparse.Commitish) (runtime.Query, error) {
	panic("NewLakeQuery called on compiler.fsCompiler")
}

func (*fsCompiler) NewLakeDeleteQuery(_ *runtime.Context, program ast.Seq, head *lakeparse.Commitish) (runtime.DeleteQuery, error) {
	panic("NewLakeDeleteQuery called on compiler.fsCompiler")
}

func optimizeAndBuild(job *Job, readers []zio.Reader) (*exec.Query, error) {
	// Call optimize to possible push down a filter predicate into the
	// kernel.Reader so that the zng scanner can do boyer-moore.
	if err := job.Optimize(); err != nil {
		return nil, err
	}
	// For an internal reader (like a shaper on intake), we don't do
	// any parallelization right now though this could be potentially
	// beneficial depending on where the bottleneck is for a given shaper.
	// See issue #2641.
	if err := job.Build(readers...); err != nil {
		return nil, err
	}
	return exec.NewQuery(job.rctx, job.Puller(), job.builder.Meter()), nil
}
