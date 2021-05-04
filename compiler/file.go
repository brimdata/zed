package compiler

import (
	"errors"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zio"
)

func CompileForFileSystem(pctx *proc.Context, p ast.Proc, reader zio.Reader, adaptor proc.DataAdaptor) (*Runtime, error) {
	runtime, err := New(pctx, p, adaptor)
	if err != nil {
		return nil, err
	}
	readers := runtime.readers
	if reader == nil {
		// If there's no reader but the DAG wants an input, then
		// flag an error.
		if len(readers) != 0 {
			return nil, errors.New("no input specified: use a command-line file or a Zed source operator")
		}
	} else {
		// If there's a reader but the DAG doesn't want an input,
		// then flag an error.
		// TBD: we could have such a configuration is a composite
		// from command includes a "pass" operator, but we can add this later.
		// See issue #2640.
		if len(readers) == 0 {
			return nil, errors.New("redundant inputs specified: use either command-line files or a Zed source operator")
		}
		if len(readers) != 1 {
			return nil, errors.New("Zed query requires a single input path")
		}
		readers[0].Reader = reader
	}
	return optimizeAndBuild(runtime)
}

func CompileJoinForFileSystem(pctx *proc.Context, p ast.Proc, readers []zio.Reader, adaptor proc.DataAdaptor) (*Runtime, error) {
	if len(readers) != 2 {
		return nil, errors.New("join operaetor requires two inputs")
	}
	runtime, err := New(pctx, p, adaptor)
	if err != nil {
		return nil, err
	}
	if len(runtime.readers) != 2 {
		return nil, errors.New("internal error: CompileJoinForFileSystem: join expected by semantic analyzer")
	}
	runtime.readers[0].Reader = readers[0]
	runtime.readers[1].Reader = readers[1]
	return optimizeAndBuild(runtime)
}

func optimizeAndBuild(runtime *Runtime) (*Runtime, error) {
	if err := runtime.Optimize(); err != nil {
		return nil, err
	}
	if err := runtime.Build(); err != nil {
		return nil, err
	}
	return runtime, nil
}
