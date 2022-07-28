package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

func newOutput(pctx *op.Context, outputs []zbuf.Puller) zbuf.Puller {
	switch len(outputs) {
	case 0:
		return nil
	case 1:
		return op.NewCatcher(op.NewSingle(outputs[0]))
	default:
		return op.NewMux(pctx, outputs)
	}
}

type compiler struct{}

// Parse concatenates the source files in filenames followed by src and parses
// the resulting program.
func (*compiler) Parse(src string, filenames ...string) (ast.Op, error) {
	return Parse(src, filenames...)
}

func (a *compiler) ParseRangeExpr(zctx *zed.Context, src string, layout order.Layout) (*zed.Value, string, error) {
	o, err := a.Parse(src)
	if err != nil {
		return nil, "", err
	}
	d, err := semantic.Analyze(context.Background(), o.(*ast.Sequential))
	if err != nil {
		return nil, "", err
	}
	if len(d.Ops) != 1 {
		return nil, "", errors.New("range expression should only have one operator")
	}
	f, ok := d.Ops[0].(*dag.Filter)
	if !ok {
		return nil, "", errors.New("range expression should be a filter")
	}
	be, ok := f.Expr.(*dag.BinaryExpr)
	if !ok {
		return nil, "", errors.New("must be a simple compare expression")
	}
	switch be.Op {
	case "<=", "<", ">=", ">":
	default:
		return nil, "", fmt.Errorf("unsupported operator: %q", be.Op)
	}
	this, ok := be.LHS.(*dag.This)
	if !ok {
		return nil, "", fmt.Errorf("left hand side must be a path")
	}
	path := field.Path(this.Path)
	if !layout.Keys.Equal(field.List{path}) {
		return nil, "", fmt.Errorf("field %q does not match pool key %q", path, layout.Keys)
	}
	val, err := querygen.EvalAtCompileTime(zctx, be.RHS) //XXX evalAtCompileTime
	if err != nil {
		return nil, "", err
	}
	return val, be.Op, nil
}

func isParallelWithLeadingFroms(o ast.Op) bool {
	par, ok := o.(*ast.Parallel)
	if !ok {
		return false
	}
	for _, o := range par.Ops {
		if !isSequentialWithLeadingFrom(o) {
			return false
		}
	}
	return true
}

func isSequentialWithLeadingFrom(o ast.Op) bool {
	seq, ok := o.(*ast.Sequential)
	if !ok && len(seq.Ops) == 0 {
		return false
	}
	_, ok = seq.Ops[0].(*ast.From)
	return ok
}

func Parse(src string, filenames ...string) (ast.Op, error) {
	parsed, err := parser.ParseZed(filenames, src)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsOp(parsed)
}

// MustParse is like Parse but panics if an error is encountered.
func MustParse(query string) ast.Op {
	o, err := (*compiler)(nil).Parse(query)
	if err != nil {
		panic(err)
	}
	return o
}
