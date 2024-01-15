package meta

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

func NewLakeMetaScanner(ctx context.Context, zctx *zed.Context, r *lake.Root, meta string) (zbuf.Scanner, error) {
	var vals []zed.Value
	var err error
	switch meta {
	case "pools":
		vals, err = r.BatchifyPools(ctx, zctx, nil)
	case "branches":
		vals, err = r.BatchifyBranches(ctx, zctx, nil)
	default:
		return nil, fmt.Errorf("unknown lake metadata type: %q", meta)
	}
	if err != nil {
		return nil, err
	}
	return zbuf.NewScanner(ctx, zbuf.NewArray(vals), nil)
}

func NewPoolMetaScanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID ksuid.KSUID, meta string) (zbuf.Scanner, error) {
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	var vals []zed.Value
	switch meta {
	case "branches":
		m := zson.NewZNGMarshalerWithContext(zctx)
		m.Decorate(zson.StylePackage)
		vals, err = p.BatchifyBranches(ctx, zctx, nil, m, nil)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown pool metadata type: %q", meta)
	}
	return zbuf.NewScanner(ctx, zbuf.NewArray(vals), nil)
}

func NewCommitMetaScanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, meta string, pruner expr.Evaluator) (zbuf.Puller, error) {
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	switch meta {
	case "objects":
		lister, err := NewSortedLister(ctx, zctx, r, p, commit, pruner)
		if err != nil {
			return nil, err
		}
		return zbuf.NewScanner(ctx, zbuf.PullerReader(lister), nil)
	case "partitions":
		lister, err := NewSortedLister(ctx, zctx, r, p, commit, pruner)
		if err != nil {
			return nil, err
		}
		slicer, err := NewSlicer(lister, zctx), nil
		if err != nil {
			return nil, err
		}
		return zbuf.NewScanner(ctx, zbuf.PullerReader(slicer), nil)
	case "log":
		tips, err := p.BatchifyBranchTips(ctx, zctx, nil)
		if err != nil {
			return nil, err
		}
		tipsScanner, err := zbuf.NewScanner(ctx, zbuf.NewArray(tips), nil)
		if err != nil {
			return nil, err
		}
		log := p.OpenCommitLog(ctx, zctx, commit)
		logScanner, err := zbuf.NewScanner(ctx, log, nil)
		if err != nil {
			return nil, err
		}
		return zbuf.MultiScanner(tipsScanner, logScanner), nil
	case "rawlog":
		reader, err := p.OpenCommitLogAsZNG(ctx, zctx, commit)
		if err != nil {
			return nil, err
		}
		return zbuf.NewScanner(ctx, reader, nil)
	case "vectors":
		snap, err := p.Snapshot(ctx, commit)
		if err != nil {
			return nil, err
		}
		vectors := commits.Vectors(snap)
		reader, err := objectReader(ctx, zctx, vectors, p.SortKey.Order)
		if err != nil {
			return nil, err
		}
		return zbuf.NewScanner(ctx, reader, nil)
	default:
		return nil, fmt.Errorf("unknown commit metadata type: %q", meta)
	}
}

func objectReader(ctx context.Context, zctx *zed.Context, snap commits.View, order order.Which) (zio.Reader, error) {
	objects := snap.Select(nil, order)
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return readerFunc(func() (*zed.Value, error) {
		if len(objects) == 0 {
			return nil, nil
		}
		val, err := m.Marshal(objects[0])
		objects = objects[1:]
		return &val, err
	}), nil
}

type readerFunc func() (*zed.Value, error)

func (r readerFunc) Read() (*zed.Value, error) { return r() }
