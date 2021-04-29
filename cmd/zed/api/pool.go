package api

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/glob"
	"github.com/brimdata/zed/pkg/units"
	"github.com/segmentio/ksuid"
)

var (
	ErrNoMatch      = errors.New("no match")
	ErrNoPoolsExist = errors.New("no pools exist")
)

type PoolCreateFlags struct {
	Keys   string
	Order  string
	Thresh units.Bytes
}

func (f *PoolCreateFlags) SetFlags(fs *flag.FlagSet) {
	f.Thresh = segment.DefaultThreshold
	fs.Var(&f.Thresh, "S", "target size of pool data objects, as '10MB' or '4GiB', etc.")
	fs.StringVar(&f.Keys, "k", "ts", "one or more pool keys to organize data in pool (cannot be changed)")
	fs.StringVar(&f.Order, "order", "desc", "sort order of newly created pool (cannot be changed)")
}

func (p *PoolCreateFlags) Create(ctx context.Context, conn *client.Connection, name string) (*api.Pool, error) {
	order, err := zedlake.ParseOrder(p.Order)
	if err != nil {
		return nil, err
	}
	keys := field.DottedList(p.Keys)
	return conn.PoolPost(ctx, api.PoolPostRequest{
		Keys:   keys,
		Name:   name,
		Order:  order,
		Thresh: int64(p.Thresh),
	})
}

func LookupPoolID(ctx context.Context, conn *client.Connection, name string) (ksuid.KSUID, error) {
	all, err := conn.PoolList(ctx)
	if err != nil {
		return ksuid.Nil, fmt.Errorf("couldn't fetch pool: %w", err)
	}
	if len(all) == 0 {
		return ksuid.Nil, ErrNoPoolsExist
	}
	for _, pool := range all {
		if pool.Name == name {
			return pool.ID, nil
		}
	}
	return ksuid.Nil, fmt.Errorf("%s: no such pool found", name)
}

func PoolGlob(ctx context.Context, conn *client.Connection, patterns ...string) ([]api.Pool, error) {
	all, err := conn.PoolList(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch pool: %w", err)
	}
	if len(all) == 0 {
		return nil, ErrNoPoolsExist
	}
	var pools []api.Pool
	if len(patterns) == 0 {
		pools = all
	} else {
		m := newPoolMap(all)
		names, err := glob.Globv(patterns, m.names())
		if err != nil {
			return nil, err
		}
		pools = m.matches(names)
		if len(pools) == 0 {
			return nil, ErrNoMatch
		}
	}
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].Name < pools[j].Name
	})
	return pools, nil
}

type poolMap map[string]api.Pool

func newPoolMap(s []api.Pool) poolMap {
	m := make(poolMap)
	for _, sp := range s {
		m[sp.Name] = sp
	}
	return m
}

func (m poolMap) names() (names []string) {
	for key := range m {
		names = append(names, key)
	}
	return
}

func (m poolMap) matches(names []string) []api.Pool {
	ss := make([]api.Pool, len(names))
	for i, name := range names {
		ss[i] = m[name]
	}
	return ss
}

func PoolNames(sl []api.Pool) []string {
	names := make([]string, 0, len(sl))
	for _, s := range sl {
		names = append(names, s.Name)
	}
	return names
}
