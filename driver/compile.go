package driver

import (
	"context"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/zap"
)

// Compile takes an AST, an input proc, and configuration parameters,
// and compiles it into a runnable flowgraph, returning a
// proc.MuxOutput that which brings together all of the flowgraphs
// tails, and is ready to be Pull()'d from.
func Compile(ctx context.Context, program ast.Proc, input proc.Proc, reverse bool, logger *zap.Logger) (*proc.MuxOutput, error) {
	pctx := &proc.Context{
		Context:     ctx,
		TypeContext: resolver.NewContext(),
		Logger:      logger,
		Warnings:    make(chan string, 5),
	}
	leaves, err := proc.CompileProc(nil, program, pctx, input)
	if err != nil {
		return nil, err
	}
	return proc.NewMuxOutput(pctx, leaves), nil
}

// LiftFilter removes the filter at the head of the flowgraph AST, if
// one is present, and returns it and the modified flowgraph AST. If
// the flowgraph does not start with a filter, it returns nil and the
// unmodified flowgraph.
func LiftFilter(p ast.Proc) (*ast.FilterProc, ast.Proc) {
	if fp, ok := p.(*ast.FilterProc); ok {
		pass := &ast.PassProc{
			Node: ast.Node{"PassProc"},
		}
		return fp, pass
	}
	seq, ok := p.(*ast.SequentialProc)
	if ok && len(seq.Procs) > 0 {
		if fp, ok := seq.Procs[0].(*ast.FilterProc); ok {
			rest := &ast.SequentialProc{
				Node:  ast.Node{"SequentialProc"},
				Procs: seq.Procs[1:],
			}
			return fp, rest
		}
	}
	return nil, p
}

// InputProc takes a Reader, optional Filter AST, and timespan, and
// constructs an input proc that can be used as the head of a
// flowgraph.
func InputProc(reader zbuf.Reader, fltast *ast.FilterProc, span nano.Span) (proc.Proc, error) {
	var f filter.Filter
	if fltast != nil {
		var err error
		if f, err = filter.Compile(fltast.Filter); err != nil {
			return nil, err
		}
	}
	return scanner.NewFilteredScanner(reader, f, span), nil
}
