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
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"
)

var MaxSpan = extent.NewGenericFromOrder(*zed.NewUint64(0), *zed.NewUint64(math.MaxUint64), order.Asc)

type Filter struct {
	zctx   *zed.Context
	engine storage.Engine
	path   *storage.URI
	expr   expr
	sem    *semaphore.Weighted
}

func NewFilter(engine storage.Engine, path *storage.URI, filter zbuf.Filter) *Filter {
	expr := compileExpr(filter.Pushdown())
	if expr == nil {
		return nil
	}
	zctx := zed.NewContext()
	return &Filter{
		zctx:   zctx,
		engine: engine,
		path:   path,
		expr:   expr,
		sem:    semaphore.NewWeighted(10),
	}
}

func (f *Filter) Apply(ctx context.Context, oid ksuid.KSUID, rules []Rule) (extent.Span, error) {
	ch := f.expr(ctx, f, oid, rules)
	if ch == nil {
		return MaxSpan, nil
	}
	r := <-ch
	return r.span, r.err
}

func (f *Filter) find(ctx context.Context, oid ksuid.KSUID, rule Rule, e dag.Expr) (extent.Span, error) {
	u := ObjectPath(f.path, rule.RuleID(), oid)
	finder, err := index.NewFinderReader(ctx, zed.NewContext(), f.engine, u, e)
	if err != nil {
		return nil, err
	}
	defer finder.Close()
	return getSpan(f.zctx, finder, rule)
}

func getSpan(zctx *zed.Context, reader zio.Reader, rule Rule) (extent.Span, error) {
	var span extent.Span
	for {
		val, err := reader.Read()
		if val == nil || err != nil {
			return span, err
		}
		min, max, err := seekMinMax(val, rule)
		if err != nil {
			return nil, err
		}
		if span == nil {
			span = extent.NewGenericFromOrder(*min, *max, order.Asc)
			continue
		}
		span.Extend(min)
		span.Extend(max)
	}
}

func seekMinMax(val *zed.Value, rule Rule) (*zed.Value, *zed.Value, error) {
	seek := rule.SeekField()
	min := val.DerefPath(field.Path{seek, "min"})
	max := val.DerefPath(field.Path{seek, "max"})
	var err error
	if min.IsError() {
		err = errors.New(zson.MustFormatValue(min))
	}
	if max.IsError() {
		err = multierr.Combine(err, errors.New(zson.MustFormatValue(max)))
	}
	return min, max, err
}
