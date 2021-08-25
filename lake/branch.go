package lake

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type BranchConfig struct {
	Name   string      `zng:"name"`
	ID     ksuid.KSUID `zng:"id"`
	Parent ksuid.KSUID `zng:"parent"`
}

type Branch struct {
	BranchConfig
	pool   *Pool
	engine storage.Engine
	log    *commit.Log
	base   commit.View
}

func newBranchConfig(name string, parent ksuid.KSUID) *BranchConfig {
	//XXX need to take parent args and check for a cycle.
	// also check that parent journal ID exists in parent.
	return &BranchConfig{
		Name:   name,
		ID:     ksuid.New(),
		Parent: parent,
	}
}

func (b *BranchConfig) Path(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(b.ID.String())
}

func (b *BranchConfig) Create(ctx context.Context, engine storage.Engine, poolPath *storage.URI, order order.Which, base journal.ID) error {
	_, err := commit.Create(ctx, engine, b.Path(poolPath), order, base)
	return err
}

func (b *BranchConfig) Open(ctx context.Context, engine storage.Engine, parent *storage.URI, pool *Pool) (*Branch, error) {
	path := b.Path(parent)
	log, err := commit.Open(ctx, engine, path, pool.Layout.Order)
	if err != nil {
		return nil, err
	}
	return &Branch{
		BranchConfig: *b,
		pool:         pool,
		engine:       engine,
		log:          log,
	}, nil
}

func (p *BranchConfig) Remove(ctx context.Context, engine storage.Engine, parent *storage.URI) error {
	//XXX unmerged data objects are left behind because they live at pool level
	return engine.DeleteByPrefix(ctx, p.Path(parent))
}

func (b *Branch) Load(ctx context.Context, r zio.Reader, date nano.Ts, author, message string) (ksuid.KSUID, error) {
	if date == 0 {
		date = nano.Now()
	}
	w, err := NewWriter(ctx, b.pool)
	if err != nil {
		return ksuid.Nil, err
	}
	err = zio.CopyWithContext(ctx, w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return ksuid.Nil, err
	}
	segments := w.Segments()
	if len(segments) == 0 {
		return ksuid.Nil, commit.ErrEmptyTransaction
	}
	id := ksuid.New()
	txn := commit.NewAddsTxn(id, w.Segments())
	txn.AppendCommitMessage(id, date, author, message)
	if _, err := b.log.Commit(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (b *Branch) Delete(ctx context.Context, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	id := ksuid.New()
	// IDs aren't vetted here and will fail at commit time if problematic.
	txn := commit.NewDeletesTxn(id, ids)
	if _, err := b.log.Commit(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (b *Branch) Merge(ctx context.Context, atTag ksuid.KSUID) (ksuid.KSUID, error) {
	var at journal.ID
	if atTag != ksuid.Nil {
		var err error
		at, err = b.log.JournalIDOfCommit(ctx, 0, atTag)
		if err != nil {
			//XXX clean up error for bad -at arg
			return ksuid.Nil, err
		}
	}
	if err := b.checkParent(ctx); err != nil {
		return ksuid.Nil, err
	}
	// To merge, we play the log to the "at" pointer onto a patch based
	// on the parent branch point.  We then create a transaction
	// from the patch's delta and try to commit this to the parent's tip.
	// We optionally squash the transaction into a single commit by allocating
	// a new commit ID for all of the merged commits.
	// If successful, we move the TAIL of this branch to one past the
	// "at" pointer (or put a nop at HEAD+1 if at points at HEAD).
	// XXX note: this NOP will turn into an UNLOCK after we add the
	// pessimistic locking.
	base := b.base
	if base == nil {
		return ksuid.Nil, errors.New("cannot merge a main branch since it has no parent")
	}
	// Play this branch's log into into the parent base patch.
	patch := commit.NewPatch(base)
	if err := patch.PlayLog(ctx, b.log, at); err != nil {
		return ksuid.Nil, err
	}
	// XXX We should have an option to keep original commit structure without
	// the squashing going on here, see isuue #2973.  In the meantime,
	// we should change the logic here to combine commit messages into the
	// squased commit, see issue #2972.
	txn := patch.NewTransaction()
	txn.AppendCommitMessage(txn.ID, nano.Now(), "<merge-operation>", "TBD: squash commit messages: see issue #2972")
	parent, err := b.openParent(ctx)
	if err != nil {
		return ksuid.Nil, err
	}
	// Commit the patch to the parent.
	newBase, err := parent.log.Commit(ctx, txn)
	if err != nil {
		return ksuid.Nil, err
	}
	// XXX Write a NOP to this branch's log so we can point the new TAIL
	// here.  This will be replaced by a locking message.  See issue #2971.
	// Hack the NOP for now with an empty commit message.
	// If we write the lock at the same point of the rebase, then writes
	// can continue just fine while the merge proceeds and the only
	// need for exclusiveity is between merge operations.
	nop := commit.NewCommitTxn(ksuid.New(), nano.Now(), "<merge-operation>", "NOP: see issue #2971", nil)
	newTip, err := b.log.Commit(ctx, nop)
	if err != nil {
		return ksuid.Nil, err
	}
	// Now update this branch's TAIL and BASE PTR so the branch is
	// grafted onto the parent at the new commit point.
	if err := b.log.MoveTail(ctx, newTip, newBase); err != nil {
		return ksuid.Nil, err
	}
	// Clear old base snapshot since it has changed.
	b.base = nil
	return txn.ID, nil
}

func (b *Branch) LookupTags(ctx context.Context, tags []ksuid.KSUID) ([]ksuid.KSUID, error) {
	var ids []ksuid.KSUID
	for _, tag := range tags {
		ok, err := b.pool.ObjectExists(ctx, tag)
		if err != nil {
			return nil, err
		}
		if ok {
			ids = append(ids, tag)
			continue
		}
		// Note that we only search in this branch not into the parent
		// since this lookup is used for indexing, deleting, etc and
		// the parent branch should be used in such cases.
		snap, ok, err := b.log.SnapshotOfCommit(ctx, 0, tag)
		if err != nil {
			return nil, fmt.Errorf("tag does not exist: %s", tag)
		}
		if !ok {
			return nil, fmt.Errorf("commit tag was previously deleted: %s", tag)
		}
		for _, seg := range snap.SelectAll() {
			ids = append(ids, seg.ID)
		}
	}
	return ids, nil
}

func (b *Branch) checkParent(ctx context.Context) error {
	var err error
	if b.Parent != ksuid.Nil && b.base == nil {
		_, basePtr, err := b.log.ReadTail(ctx)
		if err != nil {
			return err
		}
		b.base, err = b.snapParentAt(ctx, basePtr)
	}
	return err
}

// Snapshot returns a snapshot of this branch at the journal entry where
// tag is a commit.  If tag is ksuid.Nil, the snapshot is at the tip of
// the branch.
func (b *Branch) Snapshot(ctx context.Context, tag ksuid.KSUID) (commit.View, error) {
	if err := b.checkParent(ctx); err != nil {
		return nil, err
	}
	if tag == ksuid.Nil {
		// No commit reference.  Just play this branch's log
		// on top of the parent.
		if b.base == nil {
			// No parent.
			tip, err := b.log.Tip(ctx)
			if err != nil {
				if err != journal.ErrEmpty {
					return nil, err
				}
				// Nothing in this branch.  Return parent snap.
				return commit.NewSnapshot(), nil
			}
			return tip, nil
		}
		// We have a parent so create a patch and play this log
		// into into the patch.
		patch := commit.NewPatch(b.base)
		if err := patch.PlayLog(ctx, b.log, 0); err != nil {
			return nil, err
		}
		return patch, nil
	}
	// We have a commit tag.  So find the journal ID (and branch in
	// our ancesetry fort his tag.
	branch, at, err := b.searchForTag(ctx, tag)
	if err != nil {
		return nil, err
	}
	return branch.snapshotAt(ctx, at)
}

func (b *Branch) snapshotAt(ctx context.Context, at journal.ID) (commit.View, error) {
	if err := b.checkParent(ctx); err != nil {
		return nil, err
	}
	if b.base != nil {
		patch := commit.NewPatch(b.base)
		if err := patch.PlayLog(ctx, b.log, at); err != nil {
			return nil, err
		}
		return patch, nil
	}
	// No parent.  Just return the snapshot of this branch.
	return b.log.Snapshot(ctx, at)
}

func (b *Branch) searchForTag(ctx context.Context, tag ksuid.KSUID) (*Branch, journal.ID, error) {
	for {
		id, err := b.log.JournalIDOfCommit(ctx, 0, tag)
		switch err {
		case nil:
			return b, id, nil
		case commit.ErrNotFound:
			if b.Parent == ksuid.Nil {
				return nil, 0, err
			}
			b, err = b.pool.OpenBranchByID(ctx, b.Parent)
			if err != nil {
				return nil, 0, err
			}
		default:
			return nil, 0, err
		}
	}
}

func (b *Branch) openParent(ctx context.Context) (*Branch, error) {
	parent, err := b.pool.OpenBranchByID(ctx, b.Parent)
	if err != nil {
		return nil, fmt.Errorf("branch %s/%s could not access parent branch %s: %w", b.pool.Name, b.Name, b.Parent, err)
	}
	return parent, nil
}

func (b *Branch) snapParentAt(ctx context.Context, at journal.ID) (commit.View, error) {
	parent, err := b.openParent(ctx)
	if err != nil {
		return nil, err
	}
	snap, err := parent.snapshotAt(ctx, at)
	if err != nil {
		return nil, fmt.Errorf("branch %s/%s could not access parent branch %s@%d: %w", b.pool.Name, b.Name, b.Parent, at, err)
	}
	return snap, nil
}

func (b *Branch) snapshot(ctx context.Context, tag ksuid.KSUID) (commit.View, error) {
	if tag == ksuid.Nil {
		return b.log.Tip(ctx)
	}
	// XXX See issue #2955.  If commit at tag has been deleted,
	// this won't work right.
	snap, ok, err := b.log.SnapshotOfCommit(ctx, 0, tag)
	if err != nil {
		return nil, fmt.Errorf("tag does not exist: %s", tag)
	}
	if !ok {
		return nil, fmt.Errorf("commit tag was previously deleted: %s", tag)
	}
	return snap, nil
}

func (b *Branch) Pool() *Pool {
	return b.pool
}

func (b *Branch) Log() *commit.Log {
	return b.log
}

func (b *Branch) ApplyIndexRules(ctx context.Context, rules []index.Rule, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	idxrefs := make([]*index.Reference, 0, len(rules)*len(ids))
	for _, id := range ids {
		//XXX make issue for this.
		// This could be easily parallized with errgroup.
		refs, err := b.indexSegment(ctx, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		idxrefs = append(idxrefs, refs...)
	}
	id := ksuid.New()
	txn := commit.NewAddIndicesTxn(id, idxrefs)
	date := nano.Now()
	author := "indexer"
	message := index_message(rules)
	txn.AppendCommitMessage(id, date, author, message)
	if _, err := b.log.Commit(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func index_message(rules []index.Rule) string {
	skip := make(map[string]struct{})
	var names []string
	for _, r := range rules {
		name := r.RuleName()
		if _, ok := skip[name]; !ok {
			skip[name] = struct{}{}
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "index rules applied: " + strings.Join(names, ",")
}

func (b *Branch) indexSegment(ctx context.Context, rules []index.Rule, id ksuid.KSUID) ([]*index.Reference, error) {
	r, err := b.engine.Get(ctx, segment.RowObjectPath(b.pool.DataPath, id))
	if err != nil {
		return nil, err
	}
	reader := zngio.NewReader(r, zson.NewContext())
	w, err := index.NewCombiner(ctx, b.engine, b.pool.IndexPath, rules, id)
	if err != nil {
		r.Close()
		return nil, err
	}
	err = zio.CopyWithContext(ctx, w, reader)
	if err != nil {
		w.Abort()
	} else {
		err = w.Close()
	}
	if rerr := r.Close(); err == nil {
		err = rerr
	}
	return w.References(), err
}

type BranchStats struct {
	Size int64 `zng:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zng:"span"`
}

func (b *Branch) Stats(ctx context.Context, snap commit.View) (info BranchStats, err error) {
	ch := make(chan segment.Reference)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = ScanSpan(ctx, snap, nil, b.pool.Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for segment := range ch {
		info.Size += segment.RowSize
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(segment.First, segment.Last, b.pool.Layout.Order)
		} else {
			poolSpan.Extend(segment.First)
			poolSpan.Extend(segment.Last)
		}
	}
	//XXX need to change API to take return key range
	if poolSpan != nil {
		min := poolSpan.First()
		if min.Type == zng.TypeTime {
			firstTs, _ := zng.DecodeTime(min.Bytes)
			lastTs, _ := zng.DecodeTime(poolSpan.Last().Bytes)
			if lastTs < firstTs {
				firstTs, lastTs = lastTs, firstTs
			}
			span := nano.NewSpanTs(firstTs, lastTs+1)
			info.Span = &span
		}
	}
	return info, err
}
