package mock

import (
	"context"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

type Lake struct {
	poolsByName map[string]*Pool
	poolsByID   map[ksuid.KSUID]*Pool
}

var _ op.DataAdaptor = (*Lake)(nil)

type Pool struct {
	name    string
	id      ksuid.KSUID
	layout  order.Layout
	commits map[string]ksuid.KSUID
}

func NewLake() *Lake {
	return &Lake{
		poolsByName: make(map[string]*Pool),
		poolsByID:   make(map[ksuid.KSUID]*Pool),
	}
}

// create a mock lake whose key layout is embedded in the mock pool name,
// e.g., "logs-ts:asc"
func (l *Lake) NewPool(poolName string) (*Pool, error) {
	parts := strings.Split(poolName, "-")
	name := parts[0]
	layout := order.Nil
	if len(parts) > 1 {
		var err error
		layout, err = order.ParseLayout(parts[1])
		if err != nil {
			return nil, err
		}
	}
	p := &Pool{
		name:    name,
		id:      fakeID(name),
		layout:  layout,
		commits: make(map[string]ksuid.KSUID),
	}
	l.poolsByID[p.id] = p
	l.poolsByName[name] = p
	return p, nil
}

// fakeID creates a ksuid derived from the name string so we can deterministically
// check test results rather than emulating ksuid.New().
func fakeID(name string) ksuid.KSUID {
	var id ksuid.KSUID
	copy(id[:], name)
	return id
}

func (l *Lake) PoolID(_ context.Context, poolName string) (ksuid.KSUID, error) {
	pool, ok := l.poolsByName[poolName]
	if !ok {
		var err error
		pool, err = l.NewPool(poolName)
		if err != nil {
			return ksuid.Nil, err
		}
	}
	return pool.id, nil
}

func (l *Lake) CommitObject(_ context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	pool, ok := l.poolsByID[poolID]
	if !ok {
		return ksuid.Nil, fmt.Errorf("%s: no such pool", poolID)
	}
	commit, ok := pool.commits[branchName]
	if !ok {
		commit = fakeID(branchName)
		pool.commits[branchName] = commit
	}
	return commit, nil
}

func (l *Lake) Layout(_ context.Context, src dag.Source) order.Layout {
	poolSrc, ok := src.(*dag.Pool)
	if !ok {
		return order.Nil
	}
	if pool, ok := l.poolsByID[poolSrc.ID]; ok {
		return pool.layout
	}
	return order.Nil
}

func (*Lake) NewScheduler(context.Context, *zed.Context, dag.Source, extent.Span, zbuf.Filter, *dag.Filter) (op.Scheduler, error) {
	return nil, fmt.Errorf("mock.Lake.NewScheduler() should not be called")
}

func (*Lake) Open(_ context.Context, _ *zed.Context, _, _ string, _ zbuf.Filter) (zbuf.Puller, error) {
	return nil, fmt.Errorf("mock.Lake.Open() should not be called")
}
