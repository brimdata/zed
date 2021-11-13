package compiler

import (
	"errors"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zio"
)

func CompileForFileSystem(pctx *proc.Context, p ast.Proc, readers []zio.Reader, adaptor proc.DataAdaptor) (*Runtime, error) {
	runtime, err := New(pctx, p, adaptor, nil)
	if err != nil {
		return nil, err
	}
	if isJoin(p) {
		if len(readers) != 2 {
			return nil, errors.New("join operaetor requires two inputs")
		}
		if len(runtime.readers) != 2 {
			return nil, errors.New("internal error: join expected by semantic analyzer")
		}
		runtime.readers[0].Readers = readers[0:1]
		runtime.readers[1].Readers = readers[1:2]
	} else if len(readers) == 0 {
		// If there's no reader but the DAG wants an input, then
		// flag an error.
		if len(runtime.readers) != 0 {
			return nil, errors.New("no input specified: use a command-line file or a Zed source operator")
		}
	} else {
		// If there's a reader but the DAG doesn't want an input,
		// then flag an error.
		// TBD: we could have such a configuration is a composite
		// from command includes a "pass" operator, but we can add this later.
		// See issue #2640.
		if len(runtime.readers) == 0 {
			return nil, errors.New("redundant inputs specified: use either command-line files or a Zed source operator")
		}
		if len(runtime.readers) != 1 {
			return nil, errors.New("Zed query requires a single input path")
		}
		runtime.readers[0].Readers = readers
	}
	return optimizeAndBuild(runtime)
}

func isJoin(p ast.Proc) bool {
	seq, ok := p.(*ast.Sequential)
	if !ok || len(seq.Procs) == 0 {
		return false
	}
	_, ok = seq.Procs[0].(*ast.Join)
	return ok
}

func optimizeAndBuild(runtime *Runtime) (*Runtime, error) {
	// Call optimize to possible push down a filter predicate into the
	// kernel.Reader so that the zng scanner can do boyer-moore.
	if err := runtime.Optimize(); err != nil {
		return nil, err
	}
	// For an internal reader (like a shaper on intake), we don't do
	// any parallelization right now though this could be potentially
	// beneficial depending on where the bottleneck is for a given shaper.
	// See issue #2641.
	if err := runtime.Build(); err != nil {
		return nil, err
	}
	return runtime, nil
}
