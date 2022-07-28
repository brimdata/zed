package lakemanager

import (
	"context"
	"time"

	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr/extent"
)

type ObjectReader interface {
	Next() (*data.Object, error)
}

// Scan recieves a sorted stream of objects and sends to ch a series
// of Runs that are good candidates for compaction.
func Scan(ctx context.Context, reader ObjectReader, pool *pools.Config,
	thresh time.Duration, ch chan<- Run) error {
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
	cmp := extent.CompareFunc(order.Asc)
	run := NewRun(cmp)
	for {
		object, err := reader.Next()
		if object == nil {
			break
		}
		if err != nil {
			return err
		}
		// XXX An object's create timestamp is currently derived from the
		// timestamp in its ksuid ID when it should really be the commit
		// timestamp since this is when the object officially exists from the
		// lake's perspective.
		cold := time.Since(object.ID.Time()) >= thresh
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
			return err
		}
		run = NewRun(cmp)
	}
	return send(run, nil)
}
