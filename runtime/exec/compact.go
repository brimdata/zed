package exec

import (
	"context"
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/op/meta"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

func Compact(ctx context.Context, lk *lake.Root, pool *lake.Pool, branchName string, objectIDs []ksuid.KSUID, author, message, info string) (ksuid.KSUID, error) {
	if len(objectIDs) < 2 {
		return ksuid.Nil, errors.New("compact: two or more source objects required")
	}
	branch, err := pool.OpenBranchByName(ctx, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	base, err := pool.Snapshot(ctx, branch.Commit)
	if err != nil {
		return ksuid.Nil, err
	}
	compact := commits.NewSnapshot()
	for _, oid := range objectIDs {
		o, err := base.Lookup(oid)
		if err != nil {
			return ksuid.Nil, err
		}
		compact.AddDataObject(o)
	}
	zctx := zed.NewContext()
	lister := meta.NewSortedListerFromSnap(ctx, lk, pool, compact, nil)
	octx := op.NewContext(ctx, zctx, nil)
	puller := meta.NewSequenceScanner(octx, lister, pool, lister.Snapshot(), nil, nil)
	w := lake.NewSortedWriter(ctx, pool)
	if err := zbuf.CopyPuller(w, puller); err != nil {
		puller.Pull(true)
		w.Abort()
		return ksuid.Nil, err
	}
	if err := w.Close(); err != nil {
		w.Abort()
		return ksuid.Nil, err
	}
	commit, err := branch.CommitCompact(ctx, compact.SelectAll(), w.Objects(), author, message, info)
	if err != nil {
		w.Abort()
		return ksuid.Nil, err
	}
	return commit, nil
}
