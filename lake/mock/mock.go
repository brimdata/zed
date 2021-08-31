package mock

import (
	"context"
	"fmt"
	"strings"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Lake struct {
	pools map[string]Pool
}

var _ proc.DataAdaptor = (*Lake)(nil)

type Pool struct {
	name     string
	branch   string
	id       ksuid.KSUID
	branchID ksuid.KSUID
	layout   order.Layout
}

func NewLake() *Lake {
	return &Lake{
		pools: make(map[string]Pool),
	}
}

// create a mock lake whose key layout is embedded in the mock pool name,
// e.g., "logs-ts:asc"
func NewPool(poolName, branchName string) (Pool, error) {
	parts := strings.Split(poolName, "-")
	name := parts[0]
	layout := order.Nil
	if len(parts) > 1 {
		var err error
		layout, err = order.ParseLayout(parts[1])
		if err != nil {
			return Pool{}, err
		}
	}
	return Pool{
		name:     name,
		branch:   branchName,
		id:       fakeID(name),
		branchID: fakeID(branchName),
		layout:   layout,
	}, nil
}

// fakeID creates a ksuid derived from the name string so we can deterministically
// check test results rather than emulating ksuid.New().
func fakeID(name string) ksuid.KSUID {
	var id ksuid.KSUID
	copy(id[:], name)
	return id
}

func (l *Lake) IDs(_ context.Context, poolName, branchName string) (ksuid.KSUID, ksuid.KSUID, error) {
	pool, ok := l.pools[poolName]
	if !ok {
		var err error
		pool, err = NewPool(poolName, branchName)
		if err != nil {
			return ksuid.Nil, ksuid.Nil, err
		}
		l.pools[poolName] = pool
	}
	return pool.id, pool.branchID, nil
}

func (l *Lake) Layout(_ context.Context, src dag.Source) order.Layout {
	poolSrc, ok := src.(*dag.Pool)
	if !ok {
		return order.Nil
	}
	for _, pool := range l.pools {
		if pool.id == poolSrc.ID {
			return pool.layout
		}
	}
	return order.Nil
}

func (*Lake) NewScheduler(context.Context, *zson.Context, dag.Source, extent.Span, zbuf.Filter) (proc.Scheduler, error) {
	return nil, fmt.Errorf("mock.Lake.NewScheduler() should not be called")
}

func (*Lake) Open(_ context.Context, _ *zson.Context, _ string, _ zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, fmt.Errorf("mock.Lake.Open() should not be called")
}
