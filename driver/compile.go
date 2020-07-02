package driver

import (
	"context"
	"strconv"

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
func Compile(ctx context.Context, zctx *resolver.Context, program ast.Proc, reader zbuf.Reader, readerSortKey string, reverse bool, span nano.Span, logger *zap.Logger) (*MuxOutput, error) {
	ch := make(chan string, 5)
	return CompileWarningsChCustom(ctx, zctx, nil, program, reader, readerSortKey, reverse, span, logger, ch)
}

func CompileCustom(ctx context.Context, zctx *resolver.Context, custom proc.Compiler, program ast.Proc, reader zbuf.Reader, reverse bool, span nano.Span, logger *zap.Logger) (*MuxOutput, error) {
	ch := make(chan string, 5)
	return CompileWarningsChCustom(ctx, zctx, custom, program, reader, "", reverse, span, logger, ch)
}

func CompileWarningsCh(ctx context.Context, zctx *resolver.Context, program ast.Proc, reader zbuf.Reader, reverse bool, span nano.Span, logger *zap.Logger, ch chan string) (*MuxOutput, error) {
	return CompileWarningsChCustom(ctx, zctx, nil, program, reader, "", reverse, span, logger, ch)
}

func CompileWarningsChCustom(ctx context.Context, zctx *resolver.Context, custom proc.Compiler, program ast.Proc, reader zbuf.Reader, readerSortKey string, reverse bool, span nano.Span, logger *zap.Logger, ch chan string) (*MuxOutput, error) {
	ReplaceGroupByProcDurationWithKey(program)
	if readerSortKey != "" {
		dir := 1
		if reverse {
			dir = -1
		}
		setGroupByProcInputSortDir(program, readerSortKey, dir)
	}
	filterAst, program := liftFilter(program)
	scanner, err := newScanner(ctx, reader, filterAst, span)
	if err != nil {
		return nil, err
	}
	pctx := &proc.Context{
		Context:     ctx,
		TypeContext: zctx,
		Logger:      logger,
		Warnings:    ch,
	}
	leaves, err := proc.CompileProc(custom, program, pctx, scanner)
	if err != nil {
		return nil, err
	}
	return NewMuxOutput(pctx, leaves, scanner), nil
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

func ReplaceGroupByProcDurationWithKey(p ast.Proc) {
	switch p := p.(type) {
	case *ast.GroupByProc:
		if duration := p.Duration.Seconds; duration != 0 {
			durationKey := ast.ExpressionAssignment{
				Target: "ts",
				Expr: &ast.FunctionCall{
					Function: "Time.trunc",
					Args: []ast.Expression{
						&ast.FieldRead{Field: "ts"},
						&ast.Literal{
							Type:  "int64",
							Value: strconv.Itoa(duration),
						}},
				},
			}
			p.Duration.Seconds = 0
			p.Keys = append([]ast.ExpressionAssignment{durationKey}, p.Keys...)
		}
	case *ast.ParallelProc:
		for _, pp := range p.Procs {
			ReplaceGroupByProcDurationWithKey(pp)
		}
	case *ast.SequentialProc:
		for _, pp := range p.Procs {
			ReplaceGroupByProcDurationWithKey(pp)
		}
	}
}

// setGroupByProcInputSortDir examines p under the assumption that its input is
// sorted according to inputSortField and inputSortDir.  If p is an
// ast.GroupByProc and setGroupByProcInputSortDir can determine that its first
// grouping key is inputSortField or an order-preserving function of
// inputSortField, setGroupByProcInputSortDir sets ast.GroupByProc.InputSortDir
// to inputSortDir.  setGroupByProcInputSortDir returns true if it determines
// that p's output will remain sorted according to inputSortField and
// inputSortDir; otherwise, it returns false.
func setGroupByProcInputSortDir(p ast.Proc, inputSortField string, inputSortDir int) bool {
	switch p := p.(type) {
	case *ast.CutProc:
		// Return true if the output record contains inputSortField.
		for _, f := range p.Fields {
			if f == inputSortField {
				return !p.Complement
			}
		}
		return p.Complement
	case *ast.GroupByProc:
		// Set p.InputSortDir and return true if the first grouping key
		// is inputSortField or an order-preserving function of it.
		if len(p.Keys) > 0 && p.Keys[0].Target == inputSortField {
			switch expr := p.Keys[0].Expr.(type) {
			case *ast.FieldRead:
				if expr.Field == inputSortField {
					p.InputSortDir = inputSortDir
					return true
				}
			case *ast.FunctionCall:
				switch expr.Function {
				case "Math.ceil", "Math.floor", "Math.round", "Time.trunc":
					if len(expr.Args) > 0 {
						arg0, ok := expr.Args[0].(*ast.FieldRead)
						if ok && arg0.Field == inputSortField {
							p.InputSortDir = inputSortDir
							return true
						}
					}
				}
			}
		}
		return false
	case *ast.PutProc:
		for _, c := range p.Clauses {
			if c.Target == inputSortField {
				return false
			}
		}
		return true
	case *ast.SequentialProc:
		for _, pp := range p.Procs {
			if !setGroupByProcInputSortDir(pp, inputSortField, inputSortDir) {
				return false
			}
		}
		return true
	case *ast.FilterProc, *ast.HeadProc, *ast.PassProc, *ast.UniqProc, *ast.TailProc:
		return true
	default:
		return false
	}
}

// newScanner takes a Reader, optional Filter AST, and timespan, and
// constructs a scanner that can be used as the head of a
// flowgraph.
func newScanner(ctx context.Context, reader zbuf.Reader, fltast *ast.FilterProc, span nano.Span) (*scanner.Scanner, error) {
	var f filter.Filter
	if fltast != nil {
		var err error
		if f, err = filter.Compile(fltast.Filter); err != nil {
			return nil, err
		}
	}
	return scanner.NewScanner(ctx, reader, f, span), nil
}
