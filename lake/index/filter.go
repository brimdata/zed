package index

import (
	"context"
	"errors"
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	zedexpr "github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"
)

func seekDotMin(zctx *zed.Context) zedexpr.Evaluator {
	return zedexpr.NewDottedExpr(zctx, field.Dotted("seek.min"))
}

func seekDotMax(zctx *zed.Context) zedexpr.Evaluator {
	return zedexpr.NewDottedExpr(zctx, field.Dotted("seek.min"))
}

var MaxSpan = extent.NewGenericFromOrder(*zed.NewUint64(0), *zed.NewUint64(math.MaxUint64), order.Asc)

type Filter struct {
	zctx   *zed.Context
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
		zctx:   zed.NewContext(),
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
	return getSpan(f.zctx, val, finder.Order())
}

func getSpan(zctx *zed.Context, val *zed.Value, o order.Which) (extent.Span, error) {
	ectx := zedexpr.NewContext()
	min := seekDotMin(zctx).Eval(ectx, val)
	max := seekDotMax(zctx).Eval(ectx, val)
	var err error
	if min.IsError() {
		err = errors.New(zson.MustFormatValue(*min))
	}
	if max.IsError() {
		err2 := errors.New(zson.MustFormatValue(*min))
		err = multierr.Combine(err, err2)
	}
	if err != nil {
		return nil, err
	}
	return extent.NewGenericFromOrder(*min, *max, o), nil
}
