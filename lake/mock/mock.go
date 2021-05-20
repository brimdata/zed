package mock

import (
	"context"
	"fmt"
	"strings"

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
	name   string
	id     ksuid.KSUID
	layout order.Layout
}

func NewLake() *Lake {
	return &Lake{
		pools: make(map[string]Pool),
	}
}

// create a mock lake whose key layout is embedded in the mock pool name,
// e.g., "logs-ts:asc"
func NewPool(s string) (Pool, error) {
	parts := strings.Split(s, "-")
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
		name:   name,
		id:     fakeID(name),
		layout: layout,
	}, nil
}

// fakeID creates a ksuid derived from the name string so we can deterministically
// check test results rather than emulating ksuid.New().
func fakeID(name string) ksuid.KSUID {
	var id ksuid.KSUID
	copy(id[:], name)
	return id
}

func (l *Lake) Lookup(_ context.Context, name string) (ksuid.KSUID, error) {
	pool, ok := l.pools[name]
	if !ok {
		var err error
		pool, err = NewPool(name)
		if err != nil {
			return ksuid.Nil, err
		}
		l.pools[name] = pool
	}
	return pool.id, nil
}

func (l *Lake) Layout(_ context.Context, id ksuid.KSUID) (order.Layout, error) {
	for _, pool := range l.pools {
		if pool.id == id {
			return pool.layout, nil
		}
	}
	return order.Nil, fmt.Errorf("%s: no such pool", id)
}

func (*Lake) NewScheduler(context.Context, *zson.Context, ksuid.KSUID, ksuid.KSUID, extent.Span, zbuf.Filter) (proc.Scheduler, error) {
	return nil, fmt.Errorf("mock.Lake.NewScheduler() should not be called")
}

func (*Lake) Open(_ context.Context, _ *zson.Context, _ string, _ zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, fmt.Errorf("mock.Lake.Open() should not be called")
}
