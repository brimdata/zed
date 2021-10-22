package index

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
	"golang.org/x/sync/semaphore"
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

func (f *Filter) Apply(ctx context.Context, oid ksuid.KSUID, rules []Rule) (bool, error) {
	ch := f.expr(ctx, f, oid, rules)
	if ch == nil {
		return true, nil
	}
	r := <-ch
	return r.hit, r.err
}

func (f *Filter) find(ctx context.Context, oid, rid ksuid.KSUID, kv index.KeyValue, op string) (bool, error) {
	u := ObjectPath(f.path, rid, oid)
	finder, err := index.NewFinder(ctx, zed.NewContext(), f.engine, u)
	if err != nil {
		return false, err
	}
	rec, err := finder.Nearest(op, kv)
	if err != nil {
		return false, err
	}
	if rec != nil {
		return true, nil
	}
	return false, nil
}
