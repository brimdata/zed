package exec

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/extent"
)

// XXX for backward compat keep this for now, and return branchstats for pool/main
type PoolStats struct {
	Size int64 `zed:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zed:"span"`
}

func GetPoolStats(ctx context.Context, p *lake.Pool, snap commits.View) (info PoolStats, err error) {
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for _, object := range snap.Select(nil, p.Layout.Order) {
		info.Size += object.Size
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(object.First, object.Last, p.Layout.Order)
		} else {
			poolSpan.Extend(&object.First)
			poolSpan.Extend(&object.Last)
		}
	}
	//XXX need to change API to take return key range
	if poolSpan != nil {
		min := poolSpan.First()
		if min.Type == zed.TypeTime {
			firstTs := zed.DecodeTime(min.Bytes)
			lastTs := zed.DecodeTime(poolSpan.Last().Bytes)
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
	Size int64 `zed:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zed:"span"`
}

func GetBranchStats(ctx context.Context, b *lake.Branch, snap commits.View) (info BranchStats, err error) {
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for _, object := range snap.Select(nil, b.Pool().Layout.Order) {
		info.Size += object.Size
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(object.First, object.Last, b.Pool().Layout.Order)
		} else {
			poolSpan.Extend(&object.First)
			poolSpan.Extend(&object.Last)
		}
	}
	//XXX need to change API to take return key range
	if poolSpan != nil {
		min := poolSpan.First()
		if min.Type == zed.TypeTime {
			firstTs := zed.DecodeTime(min.Bytes)
			lastTs := zed.DecodeTime(poolSpan.Last().Bytes)
			if lastTs < firstTs {
				firstTs, lastTs = lastTs, firstTs
			}
			span := nano.NewSpanTs(firstTs, lastTs+1)
			info.Span = &span
		}
	}
	return info, err
}
