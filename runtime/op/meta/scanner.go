package meta

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

func NewLakeMetaScanner(ctx context.Context, zctx *zed.Context, r *lake.Root, meta string, filter zbuf.Filter) (zbuf.Scanner, error) {
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
	return zbuf.MultiScanner(s), nil
}

func NewPoolMetaScanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID ksuid.KSUID, meta string, filter zbuf.Filter) (zbuf.Scanner, error) {
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
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown pool metadata type: %q", meta)
	}
	s, err := zbuf.NewScanner(ctx, zbuf.NewArray(vals), filter)
	if err != nil {
		return nil, err
	}
	return zbuf.MultiScanner(s), nil
}

func NewCommitMetaScanner(ctx context.Context, zctx *zed.Context, r *lake.Root, poolID, commit ksuid.KSUID, meta string, filter zbuf.Filter) (zbuf.Scanner, error) {
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
		return zbuf.MultiScanner(s), nil
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
		return zbuf.MultiScanner(s), nil
	case "partitions":
		snap, err := p.Snapshot(ctx, commit)
		if err != nil {
			return nil, err
		}
		reader, err := partitionReader(ctx, zctx, p.Layout, snap, filter)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return zbuf.MultiScanner(s), nil
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
		return zbuf.MultiScanner(tipsScanner, logScanner), nil
	case "rawlog":
		reader, err := p.OpenCommitLogAsZNG(ctx, zctx, commit)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return zbuf.MultiScanner(s), nil
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
		return zbuf.MultiScanner(s), nil
	default:
		return nil, fmt.Errorf("unknown commit metadata type: %q", meta)
	}
}

func objectReader(ctx context.Context, zctx *zed.Context, snap commits.View, order order.Which) (zio.Reader, error) {
	ch := make(chan data.Object)
	ctx, cancel := context.WithCancel(ctx)
	var scanErr error
	go func() {
		scanErr = Scan(ctx, snap, order, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	return readerFunc(func() (*zed.Value, error) {
		select {
		case p := <-ch:
			if p.ID == ksuid.Nil {
				cancel()
				return nil, scanErr
			}
			rec, err := m.Marshal(p)
			if err != nil {
				cancel()
				return nil, err
			}
			return rec, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}), nil
}
