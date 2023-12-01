package compiler

import (
	"errors"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zio"
)

type fsCompiler struct {
	anyCompiler
	src *data.Source
}

func NewFileSystemCompiler(engine storage.Engine) runtime.Compiler {
	return &fsCompiler{src: data.NewSource(engine, nil)}
}

func (f *fsCompiler) NewQuery(octx *op.Context, seq ast.Seq, readers []zio.Reader) (*runtime.Query, error) {
	job, err := NewJob(octx, seq, f.src, nil)
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

func (*fsCompiler) NewLakeQuery(octx *op.Context, program ast.Seq, parallelism int, head *lakeparse.Commitish) (*runtime.Query, error) {
	panic("NewLakeQuery called on compiler.fsCompiler")
}

func (*fsCompiler) NewLakeDeleteQuery(octx *op.Context, program ast.Seq, head *lakeparse.Commitish) (*runtime.DeleteQuery, error) {
	panic("NewLakeDeleteQuery called on compiler.fsCompiler")
}

func optimizeAndBuild(job *Job, readers []zio.Reader) (*runtime.Query, error) {
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
	return runtime.NewQuery(job.octx, job.Puller(), job.builder.Meter()), nil
}
