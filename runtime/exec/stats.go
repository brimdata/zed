package exec

import (
	"context"

	"github.com/superdb/super"
	"github.com/superdb/super/db"
	"github.com/superdb/super/db/commits"
	"github.com/superdb/super/order"
	"github.com/superdb/super/pkg/nano"
	"github.com/superdb/super/runtime/sam/expr/extent"
)

// XXX for backward compat keep this for now, and return branchstats for table/main
type TableStats struct {
	Size int64 `super:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `super:"span"`
}

func GetTableStats(ctx context.Context, t *db.Table, snap commits.View) (info TableStats, err error) {
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var tableSpan *extent.Generic
	sortKey, ok := t.SortKeys.Primary()
	if !ok {
		for _, object := range snap.Select(nil, order.Asc) {
			info.Size += object.Size
		}
	} else {
		for _, object := range snap.Select(nil, sortKey.Order) {
			info.Size += object.Size
			if tableSpan == nil {
				tableSpan = extent.NewGenericFromOrder(object.Min, object.Max, order.Asc)
			} else {
				tableSpan.Extend(object.Min)
				tableSpan.Extend(object.Max)
			}
		}
	}
	//XXX need to change API to take return key range
	if tableSpan != nil {
		min := tableSpan.First()
		if min.Type() == super.TypeTimestamp {
			firstTs := super.DecodeTime(min.Bytes())
			lastTs := super.DecodeTime(tableSpan.Last().Bytes())
			if lastTs < firstTs {
				firstTs, lastTs = lastTs, firstTs
			}
			span := nano.NewSpanTs(firstTs, lastTs+1)
			info.Span = &span
		}
	}
	return info, err
}

type BranchStats struct {
	Size int64 `super:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `super:"span"`
}

func GetBranchStats(ctx context.Context, b *db.Branch, snap commits.View) (info BranchStats, err error) {
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var tableSpan *extent.Generic
	o := order.Asc
	if sortKey, ok := b.Table().SortKeys.Primary(); ok {
		o = sortKey.Order
	}
	for _, object := range snap.Select(nil, o) {
		info.Size += object.Size
		if tableSpan == nil {
			tableSpan = extent.NewGenericFromOrder(object.Min, object.Max, order.Asc)
		} else {
			tableSpan.Extend(object.Min)
			tableSpan.Extend(object.Max)
		}
	}
	//XXX need to change API to take return key range
	if tableSpan != nil {
		min := tableSpan.First()
		if min.Type() == super.TypeTimestamp {
			firstTs := super.DecodeTime(min.Bytes())
			lastTs := super.DecodeTime(tableSpan.Last().Bytes())
			if lastTs < firstTs {
				firstTs, lastTs = lastTs, firstTs
			}
			span := nano.NewSpanTs(firstTs, lastTs+1)
			info.Span = &span
		}
	}
	return info, err
}
