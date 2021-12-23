package index

import (
	"context"
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	zedexpr "github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"
)

var (
	minExpr = zedexpr.NewDottedExpr(field.Dotted("seek.min"))
	maxExpr = zedexpr.NewDottedExpr(field.Dotted("seek.max"))
	MaxSpan = extent.NewGenericFromOrder(*zed.NewUint64(0), *zed.NewUint64(math.MaxUint64), order.Asc)
)

type Filter struct {
	engine storage.Engine
	path   *storage.URI
	expr   expr
	sem    *semaphore.Weighted
}

func NewFilter(engine storage.Engine, path *storage.URI, dag *dag.Filter) (*Filter, error) {
	expr, err := compileExpr(dag.Expr)
	if err != nil {
		return nil, err
	}
	return &Filter{
		engine: engine,
		path:   path,
		expr:   expr,
		sem:    semaphore.NewWeighted(10),
	}, nil
}

func (f *Filter) Apply(ctx context.Context, oid ksuid.KSUID, rules []Rule) (extent.Span, error) {
	ch := f.expr(ctx, f, oid, rules)
	if ch == nil {
		return MaxSpan, nil
	}
	r := <-ch
	return r.span, r.err
}

func (f *Filter) find(ctx context.Context, oid, rid ksuid.KSUID, kv index.KeyValue, op string) (extent.Span, error) {
	u := ObjectPath(f.path, rid, oid)
	finder, err := index.NewFinder(ctx, zed.NewContext(), f.engine, u)
	if err != nil {
		return nil, err
	}
	val, err := finder.Nearest(op, kv)
	if val == nil || err != nil {
		return nil, err
	}
	return getSpan(val, finder.Order())
}

func getSpan(val *zed.Value, o order.Which) (extent.Span, error) {
	ectx := zedexpr.NewContext()
	min := minExpr.Eval(ectx, val)
	max := maxExpr.Eval(ectx, val)
	var err error
	if min.Type == zed.TypeError {
		err, _ = zed.DecodeError(min.Bytes)
	}
	if max.Type == zed.TypeError {
		err2, _ := zed.DecodeError(max.Bytes)
		err = multierr.Combine(err, err2)
	}
	if err != nil {
		return nil, err
	}
	return extent.NewGenericFromOrder(*min, *max, o), nil
}
