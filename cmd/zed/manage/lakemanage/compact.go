package lakemanage

import (
	"context"

	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

// CompactionScan recieves a sorted stream of objects and sends to ch a series
// of Runs that are good candidates for compaction.
func CompactionScan(ctx context.Context, it DataObjectIterator, pool *pools.Config,
	ch chan<- Run) error {
	send := func(run Run) error {
		if len(run.Objects) > 1 {
			select {
			case ch <- run:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	cmp := expr.NewValueCompareFn(order.Asc, true)
	run := NewRun(cmp)
	for {
		object, err := it.Next()
		if object == nil {
			return send(run)
		}
		if err != nil {
			return err
		}
		if run.Overlaps(&object.Min, &object.Max) || object.Size < pool.Threshold/2 {
			run.Add(object)
			continue
		}
		if err := send(run); err != nil {
			return err
		}
		run = NewRun(cmp)
		run.Add(object)
	}
}

type PoolDataObjectIterator struct {
	reader      zio.ReadCloser
	unmarshaler *zson.UnmarshalZNGContext
}

func NewPoolDataObjectIterator(ctx context.Context, lake api.Interface, head *lakeparse.Commitish,
	sortKey order.SortKey) (*PoolDataObjectIterator, error) {
	query, err := head.FromSpec("objects")
	if err != nil {
		return nil, err
	}
	if sortKey.Order == order.Asc {
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
