package lakemanage

import (
	"context"
	"time"

	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

// CompactionScan recieves a sorted stream of objects and sends to ch a series
// of Runs that are good candidates for compaction. If there are hot objects
// in the pool, CompactionScan returns the timestamp when the next object turns cool,
// otherwise nil.
func CompactionScan(ctx context.Context, it DataObjectIterator, pool *pools.Config,
	thresh time.Duration, ch chan<- Run) (*time.Time, error) {
	send := func(run Run, next extent.Span) error {
		// Send a run if it contains more than one object and the total size of
		// objects unobscured by the next span is greater than at least half
		// of the pool threshold.
		if len(run.Objects) > 1 && run.SizeUncoveredBy(next) > pool.Threshold/2 {
			select {
			case ch <- run:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	var nextcold *time.Time
	cmp := extent.CompareFunc(order.Asc)
	run := NewRun(cmp)
	for {
		object, err := it.Next()
		if object == nil {
			break
		}
		if err != nil {
			return nil, err
		}
		// XXX An object's create timestamp is currently derived from the
		// timestamp in its ksuid ID when it should really be the commit
		// timestamp since this is when the object officially exists from the
		// lake's perspective.
		ts := object.ID.Time()
		cold := time.Since(ts) >= thresh
		if !cold {
			coldtime := ts.Add(thresh)
			if nextcold == nil || (*nextcold).After(coldtime) {
				nextcold = &coldtime
			}
		}
		// There's two cases we are concerned with:
		// 1. Reduction of overlapping objects
		// 2. Consolidating patches of small objects into larger single blocks.
		// add object to current run if it overlaps *or* object size is less than
		// a quarter of thresh.
		if cold && (object.Size <= pool.Threshold/4 || run.Overlaps(&object.First, &object.Last)) {
			run.Add(object)
			continue
		}
		if err := send(run, object.Span(order.Asc)); err != nil {
			return nil, err
		}
		run = NewRun(cmp)
	}
	return nextcold, send(run, nil)
}

type PoolDataObjectIterator struct {
	reader      zio.ReadCloser
	unmarshaler *zson.UnmarshalZNGContext
}

func NewPoolDataObjectIterator(ctx context.Context, lake api.Interface, head *lakeparse.Commitish,
	layout order.Layout) (*PoolDataObjectIterator, error) {
	query, err := head.FromSpec("objects")
	if err != nil {
		return nil, err
	}
	if layout.Order == order.Asc {
		query += " | sort meta.first"
	} else {
		query += " | sort meta.last"
	}
	r, err := lake.Query(ctx, nil, query)
	if err != nil {
		return nil, err
	}
	return &PoolDataObjectIterator{
		reader:      r,
		unmarshaler: zson.NewZNGUnmarshaler(),
	}, nil
}

func (r *PoolDataObjectIterator) Next() (*data.Object, error) {
	val, err := r.reader.Read()
	if val == nil || err != nil {
		return nil, err
	}
	var o data.Object
	if err := r.unmarshaler.Unmarshal(val, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *PoolDataObjectIterator) Close() error {
	return r.reader.Close()
}

type DataObjectIterator interface {
	Next() (*data.Object, error)
}
