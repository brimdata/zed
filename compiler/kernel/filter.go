package kernel

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
)

type Filter struct {
	Pushdown dag.Expr
	builder  *Builder
}

var _ zbuf.Filter = (*Filter)(nil)

func (f *Filter) AsEvaluator() (expr.Evaluator, error) {
	if f == nil {
		return nil, nil
	}
	return f.builder.compileExpr(f.Pushdown)
}

func (f *Filter) AsBufferFilter() (*expr.BufferFilter, error) {
	if f == nil {
		return nil, nil
	}
	return CompileBufferFilter(f.builder.pctx.Zctx, f.Pushdown)
}

func (f *Filter) AsObjectFilter(o order.Which, key field.Path) (*expr.ObjectFilter, error) {
	if f == nil {
		return nil, nil
	}
	pk := NewPoolKeyFilter(key, f.Pushdown)
	if pk == nil {
		return nil, nil
	}
	e := pk.ObjectFilterExpr(o)
	eval, err := CompileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewObjectFilter(eval), nil
}

func (f *Filter) AsObjectTruncatedFilter(o order.Which, key field.Path) (*expr.ObjectFilter, error) {
	if f == nil {
		return nil, nil
	}
	pk := NewPoolKeyFilter(key, f.Pushdown)
	if pk == nil {
		return nil, nil
	}
	e := pk.ObjectTruncatedExpr(o)
	eval, err := CompileExpr(e)
	if err != nil {
		return nil, err
	}
	return expr.NewObjectFilter(eval), nil
}

func (f *Filter) PoolKeyFilter(key field.Path) *PoolKeyFilter {
	if f == nil {
		return nil
	}
	return NewPoolKeyFilter(key, f.Pushdown)
}

type PoolKeyFilter struct {
	Expr dag.Expr
}

func NewPoolKeyFilter(key field.Path, node dag.Expr) *PoolKeyFilter {
	var p PoolKeyFilter
	p.Expr = p.walk(node, func(cmp string, lhs *dag.This, rhs *dag.Literal) dag.Expr {
		if !key.Equal(lhs.Path) {
			return nil
		}
		return &dag.BinaryExpr{
			Op:  cmp,
			LHS: &dag.This{Path: []string{"key"}},
			RHS: rhs,
		}
	})
	if p.Expr == nil {
		return nil
	}
	return &p
}

func (p *PoolKeyFilter) ObjectFilterExpr(o order.Which, prefix ...string) dag.Expr {
	lower := append([]string{}, append(prefix, "lower")...)
	upper := append([]string{}, append(prefix, "upper")...)
	return p.Walk(func(cmp string, this *dag.This, val *dag.Literal) dag.Expr {
		switch cmp {
		case "==":
			return &dag.BinaryExpr{
				Op:  "and",
				LHS: compare("<=", &dag.This{Path: lower}, val, o),
				RHS: compare(">=", &dag.This{Path: upper}, val, o),
			}
		case "<", "<=":
			this.Path = lower
		case ">", ">=":
			this.Path = upper
		}
		return compare(cmp, this, val, o)
	})
}

func compare(op string, lhs, rhs dag.Expr, o order.Which) *dag.BinaryExpr {
	nullsMax := &dag.Literal{Value: "false"}
	if o == order.Asc {
		nullsMax.Value = "true"
	}
	return &dag.BinaryExpr{
		Op: op,
		LHS: &dag.Call{
			Name: "compare",
			Args: []dag.Expr{lhs, rhs, nullsMax},
		},
		RHS: &dag.Literal{Value: "0"},
	}
}

func (p *PoolKeyFilter) ObjectTruncatedExpr(o order.Which) dag.Expr {
	return p.Walk(func(cmp string, this *dag.This, val *dag.Literal) dag.Expr {
		switch cmp {
		case "==":
			return &dag.Literal{Value: "false"}
		case "<", "<=":
			this.Path = []string{"upper"}
		case ">", ">=":
			this.Path = []string{"lower"}
		}
		return compare(cmp, this, val, o)
	})
}

type visit func(cmp string, lhs *dag.This, rhs *dag.Literal) dag.Expr

func (p *PoolKeyFilter) Walk(v func(cmp string, lhs *dag.This, rhs *dag.Literal) dag.Expr) dag.Expr {
	return p.walk(p.Expr, v)
}

func (p *PoolKeyFilter) walk(node dag.Expr, v func(cmp string, lhs *dag.This, rhs *dag.Literal) dag.Expr) dag.Expr {
	e, ok := node.(*dag.BinaryExpr)
	if !ok {
		return nil
	}
	switch e.Op {
	case "or", "and":
		lhs := p.walk(e.LHS, v)
		rhs := p.walk(e.RHS, v)
		if lhs == nil {
			return rhs
		}
		if rhs == nil {
			return lhs
		}
		return &dag.BinaryExpr{
			Op:  e.Op,
			LHS: lhs,
			RHS: rhs,
		}
	case "==", "<", "<=", ">", ">=":
		this, ok := e.LHS.(*dag.This)
		if !ok {
			return nil
		}
		rhs, ok := e.RHS.(*dag.Literal)
		if !ok {
			return nil
		}
		// Copy this.
		var lhs dag.This
		lhs.Path = append(lhs.Path, this.Path...)
		return v(e.Op, &lhs, rhs)
	default:
		return nil
	}
}
