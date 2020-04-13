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

// Compile takes an AST, an input reader, and configuration parameters,
// and compiles it into a runnable flowgraph, returning a
// proc.MuxOutput that which brings together all of the flowgraphs
// tails, and is ready to be Pull()'d from.
func Compile(ctx context.Context, program ast.Proc, reader zbuf.Reader, reverse bool, span nano.Span, logger *zap.Logger) (*MuxOutput, error) {

	filterAst, program := liftFilter(program)
	input, err := inputProc(reader, filterAst, span)
	if err != nil {
		return nil, err
	}
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
	return NewMuxOutput(pctx, leaves), nil
}

// liftFilter removes the filter at the head of the flowgraph AST, if
// one is present, and returns it and the modified flowgraph AST. If
// the flowgraph does not start with a filter, it returns nil and the
// unmodified flowgraph.
func liftFilter(p ast.Proc) (*ast.FilterProc, ast.Proc) {
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

// inputProc takes a Reader, optional Filter AST, and timespan, and
// constructs an input proc that can be used as the head of a
// flowgraph.
func inputProc(reader zbuf.Reader, fltast *ast.FilterProc, span nano.Span) (proc.Proc, error) {
	var f filter.Filter
	if fltast != nil {
		var err error
		if f, err = filter.Compile(fltast.Filter); err != nil {
			return nil, err
		}
	}
	return scanner.NewScanner(reader, f, span), nil
}
