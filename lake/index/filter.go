package index

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

type Filter struct {
	engine  storage.Engine
	path    *storage.URI
	matcher matcher
}

func NewFilter(engine storage.Engine, path *storage.URI, dag []dag.IndexPredicate) (*Filter, error) {
	// XXX The matcher closure has a memoization map in state, this needs to be
	// fixed so this will work with concurrency.
	matcher, err := compilePredicates(dag)
	if err != nil {
		return nil, err
	}
	return &Filter{
		engine:  engine,
		path:    path,
		matcher: matcher,
	}, nil
}

func (f *Filter) Apply(ctx context.Context, oid ksuid.KSUID, rules []Rule) (bool, error) {
	for _, match := range f.matcher(rules) {
		hit, err := f.find(ctx, oid, match)
		if hit || err != nil {
			return hit, err
		}
	}
	return false, nil
}

func (f *Filter) find(ctx context.Context, oid ksuid.KSUID, match match) (bool, error) {
	u := ObjectPath(f.path, match.rule.RuleID(), oid)
	finder, err := index.NewFinder(ctx, zed.NewContext(), f.engine, u)
	if err != nil {
		return false, err
	}
	for _, k := range match.lookupKeys {
		rec, err := finder.Lookup(k)
		if err != nil {
			return false, err
		}
		if rec != nil {
			return true, nil
		}
	}
	return false, nil
}
