package exec

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/op/from"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

//XXX for backward compat keep this for now, and return branchstats for pool/main
type PoolStats struct {
	Size int64 `zed:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zed:"span"`
}

func GetPoolStats(ctx context.Context, p *lake.Pool, snap commits.View) (info PoolStats, err error) {
	ch := make(chan data.Object)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = Scan(ctx, snap, p.Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for object := range ch {
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
	ch := make(chan data.Object)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = Scan(ctx, snap, b.Pool().Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for object := range ch {
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

func NewLakeMetaPlanner(ctx context.Context, zctx *zed.Context, r *lake.Root, meta string, filter zbuf.Filter) (from.Planner, error) {
	f, err := filter.AsEvaluator()
	if err != nil {
		return nil, err
	}
	var vals []zed.Value
	switch meta {
	case "pools":
		vals, err = r.BatchifyPools(ctx, zctx, f)
	case "branches":
		vals, err = r.BatchifyBranches(ctx, zctx, f)
	case "index_rules":
		vals, err = r.BatchifyIndexRules(ctx, zctx, f)
	default:
		return nil, fmt.Errorf("unknown lake metadata type: %q", meta)
	}
	if err != nil {
		return nil, err
	}
	s, err := zbuf.NewScanner(ctx, zbuf.NewArray(vals), filter)
	if err != nil {
		return nil, err
	}
	return newScannerScheduler(s), nil
}

func NewPoolMetaPlanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID ksuid.KSUID, meta string, filter zbuf.Filter) (from.Planner, error) {
	f, err := filter.AsEvaluator()
	if err != nil {
		return nil, err
	}
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	var vals []zed.Value
	switch meta {
	case "branches":
		m := zson.NewZNGMarshalerWithContext(zctx)
		m.Decorate(zson.StylePackage)
		vals, err = p.BatchifyBranches(ctx, zctx, nil, m, f)
	default:
		return nil, fmt.Errorf("unknown pool metadata type: %q", meta)
	}
	s, err := zbuf.NewScanner(ctx, zbuf.NewArray(vals), filter)
	if err != nil {
		return nil, err
	}
	return newScannerScheduler(s), nil
}

func NewCommitMetaPlanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, meta string, filter zbuf.Filter) (from.Planner, error) {
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	switch meta {
	case "objects":
		snap, err := p.Snapshot(ctx, commit)
		if err != nil {
			return nil, err
		}
		reader, err := objectReader(ctx, zctx, snap, p.Layout.Order)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "indexes":
		snap, err := p.Snapshot(ctx, commit)
		if err != nil {
			return nil, err
		}
		reader, err := indexObjectReader(ctx, zctx, snap, p.Layout.Order)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "partitions":
		snap, err := p.Snapshot(ctx, commit)
		if err != nil {
			return nil, err
		}
		reader, err := partitionReader(ctx, zctx, p.Layout, snap)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "log":
		tips, err := p.BatchifyBranchTips(ctx, zctx, nil)
		if err != nil {
			return nil, err
		}
		tipsScanner, err := zbuf.NewScanner(ctx, zbuf.NewArray(tips), filter)
		if err != nil {
			return nil, err
		}
		log := p.OpenCommitLog(ctx, zctx, commit)
		logScanner, err := zbuf.NewScanner(ctx, log, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(tipsScanner, logScanner), nil
	case "rawlog":
		reader, err := p.OpenCommitLogAsZNG(ctx, zctx, commit)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "vectors":
		snap, err := p.Snapshot(ctx, commit)
		if err != nil {
			return nil, err
		}
		vectors := commits.Vectors(snap)
		reader, err := objectReader(ctx, zctx, vectors, p.Layout.Order)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	default:
		return nil, fmt.Errorf("unknown commit metadata type: %q", meta)
	}
}
