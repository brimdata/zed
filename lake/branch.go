package lake

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	maxCommitRetries  = 10
	maxMessageObjects = 10
)

var ErrCommitFailed = fmt.Errorf("exceeded max update attempts (%d) to branch tip: commit failed", maxCommitRetries)

type Branch struct {
	branches.Config
	pool   *Pool
	engine storage.Engine
	//base   commits.View
}

func OpenBranch(ctx context.Context, config *branches.Config, engine storage.Engine, poolPath *storage.URI, pool *Pool) (*Branch, error) {
	return &Branch{
		Config: *config,
		pool:   pool,
		engine: engine,
	}, nil
}

func (b *Branch) Load(ctx context.Context, r zio.Reader, author, message string) (ksuid.KSUID, error) {
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
	objects := w.Objects()
	if len(objects) == 0 {
		return ksuid.Nil, commits.ErrEmptyTransaction
	}
	if message == "" {
		message = loadMessage(objects)
	}
	// The load operation has only added new objects so we know its
	// safe to merge at the tip and there can be no conflicts
	// with other concurrent writers (except for updating the branch pointer
	// which is handled by Branch.commit)
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		return commits.NewAddsObject(parent.Commit, retries, author, message, objects), nil
	})
}

func loadMessage(objects []data.Object) string {
	var b strings.Builder
	plural := "s"
	if len(objects) == 1 {
		plural = ""
	}
	b.WriteString(fmt.Sprintf("loaded %d data object%s\n\n", len(objects), plural))
	for k, o := range objects {
		b.WriteString("  ")
		b.WriteString(o.String())
		b.WriteByte('\n')
		if k >= maxMessageObjects {
			b.WriteString("  ...\n")
			break
		}
	}
	return b.String()
}

func (b *Branch) Delete(ctx context.Context, ids []ksuid.KSUID, author, message string) (ksuid.KSUID, error) {
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		snap, err := b.pool.commits.Snapshot(ctx, parent.Commit)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if !snap.Exists(id) {
				return nil, fmt.Errorf("non-existent object %s: delete operation aborted", id)
			}
		}
		return commits.NewDeletesObject(parent.Commit, retries, author, message, ids), nil
	})
}

func (b *Branch) Revert(ctx context.Context, commit ksuid.KSUID, author, message string) (ksuid.KSUID, error) {
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		patch, err := b.pool.commits.PatchOfCommit(ctx, commit)
		if err != nil {
			return nil, fmt.Errorf("commit not found: %s", commit)
		}
		tip, err := b.pool.commits.Snapshot(ctx, parent.Commit)
		if err != nil {
			return nil, err
		}
		if message == "" {
			message = fmt.Sprintf("reverted commit %s", commit)
		}
		object, err := patch.Revert(tip, ksuid.New(), parent.Commit, retries, author, message)
		if err != nil {
			return nil, err
		}
		return object, nil
	})
}

func (b *Branch) mergeInto(ctx context.Context, parent *Branch, author, message string) (ksuid.KSUID, error) {
	if b == parent {
		return ksuid.Nil, errors.New("cannot merge branch into itself")
	}
	return parent.commit(ctx, func(head *branches.Config, retries int) (*commits.Object, error) {
		return b.buildMergeObject(ctx, head, retries, author, message, parent.Name)
	})
	//XXX we should follow parent commit with a child rebase... do this
	// next... we want to fast forward the child to any pending commits
	// that happened on the child branch while we were merging into the parent.
	// and rebase the child branch to point at the parent where we grafted on
	// it's ok if new commits are arriving past the parent graft on point...
}

func (b *Branch) buildMergeObject(ctx context.Context, parent *branches.Config, retries int, author, message, parentName string) (*commits.Object, error) {
	childPath, err := b.pool.commits.Path(ctx, b.Commit)
	if err != nil {
		return nil, err
	}
	parentPath, err := b.pool.commits.Path(ctx, parent.Commit)
	if err != nil {
		return nil, err
	}
	baseID := commonAncestor(parentPath, childPath)
	if baseID == ksuid.Nil {
		//XXX this shouldn't happen because because all of the branches
		// should live in a single tree.
		//XXX hmm, except if you branch main when it is empty...?
		// we shoudl detect this and not allow it...?
		return nil, errors.New("system error: cannot locate common ancestor for branch merge")
	}
	// Compute the snapshot of the common ancestor then compute patches
	// along each branch and make sure the two patches do not have a
	// delete conflict.  For now, this is the only kind of merge update
	// conflict we detect.
	base, err := b.pool.commits.Snapshot(ctx, baseID)
	if err != nil {
		return nil, err
	}
	childPatch, err := b.pool.commits.PatchOfPath(ctx, base, baseID, b.Commit)
	if err != nil {
		return nil, err
	}
	parentPatch, err := b.pool.commits.PatchOfPath(ctx, base, baseID, parent.Commit)
	if err != nil {
		return nil, err
	}
	if overlap := childPatch.OverlappingDeletes(parentPatch); overlap != nil {
		//XXX add IDs of (some of the) overlaps
		return nil, errors.New("write conflict on merge")
	}
	if message == "" {
		message = fmt.Sprintf("merged %s into %s", b.Name, parent.Name)
	}
	return childPatch.NewCommitObject(parent.Commit, retries, author, message), nil
}

func commonAncestor(a, b []ksuid.KSUID) ksuid.KSUID {
	m := make(map[ksuid.KSUID]struct{})
	for _, id := range a {
		m[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := m[id]; ok {
			return id
		}
	}
	return ksuid.Nil
}

type constructor func(parent *branches.Config, retries int) (*commits.Object, error)

func (b *Branch) commit(ctx context.Context, create constructor) (ksuid.KSUID, error) {
	// A commit must append new state to the tip of the branch while simultaneously
	// upating the branch pointer in a trasactionally consistent fashion.
	// For example, if we compute a commit object based on a certain tip commit,
	// then commit that object after another writer commits in between,
	// the commit object may be inconsistent against the intervening commit.
	//
	// We do this update optimistically and ensure this consistency with
	// a loop that builds the commit object based on the presumed parent,
	// then moves the branch pointer to the new commit but, using a constraint,
	// only succeeds when the presumed parent is atomically consistent
	// with the branch update.  If the contraint, fails will loop a number
	// of times till it succeeds, or we give up.
	for retries := 0; retries < maxCommitRetries; retries++ {
		config, err := b.pool.branches.LookupByName(ctx, b.Name)
		if err != nil {
			return ksuid.Nil, err
		}
		object, err := create(config, retries)
		if err != nil {
			return ksuid.Nil, err
		}
		if err := b.pool.commits.Put(ctx, object); err != nil {
			return ksuid.Nil, fmt.Errorf("branch %q failed to write commit object: %w", b.Name, err)
		}
		// Set the branch pointer to point to this commit object
		// and stash the current commit (that will become the parent)
		// in a local for the constraint check closure.
		parent := config.Commit
		config.Commit = object.Commit
		parentCheck := func(e journal.Entry) bool {
			if entry, ok := e.(*branches.Config); ok {
				return entry.Commit == parent
			}
			return false
		}
		if err := b.pool.branches.Update(ctx, config, parentCheck); err != nil {
			if err == journal.ErrConstraint {
				// Parent check failed so try again.
				if err := b.pool.commits.Remove(ctx, object); err != nil {
					return ksuid.Nil, err
				}
				continue
			}
			return ksuid.Nil, err
		}
		return object.Commit, nil
	}
	return ksuid.Nil, fmt.Errorf("branch %q: %w", b.Name, ErrCommitFailed)
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
		patch, err := b.pool.commits.PatchOfCommit(ctx, tag)
		if err != nil {
			continue
		}
		ids = append(ids, patch.DataObjects()...)
	}
	return ids, nil
}

func (b *Branch) Pool() *Pool {
	return b.pool
}

func (b *Branch) ApplyIndexRules(ctx context.Context, rules []index.Rule, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	idxrefs := make([]*index.Object, 0, len(rules)*len(ids))
	for _, id := range ids {
		//XXX make issue for this.
		// This could be easily parallized with errgroup.
		refs, err := b.indexObject(ctx, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		idxrefs = append(idxrefs, refs...)
	}
	author := "indexer"
	message := index_message(rules)
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		return commits.NewAddIndexesObject(parent.Commit, author, message, retries, idxrefs), nil
	})
}

func (b *Branch) UpdateIndex(ctx context.Context, rules []index.Rule) (ksuid.KSUID, error) {
	snap, err := b.pool.commits.Snapshot(ctx, b.Commit)
	if err != nil {
		return ksuid.Nil, err
	}
	var objects []*index.Object
	for id, rules := range snap.Unindexed(rules) {
		o, err := b.indexObject(ctx, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		objects = append(objects, o...)
	}
	if len(objects) == 0 {
		return ksuid.Nil, errors.New("indices are up to date")
	}
	const author = "indexer"
	message := index_message(rules)
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		return commits.NewAddIndexesObject(parent.Commit, author, message, retries, objects), nil
	})
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

func (b *Branch) indexObject(ctx context.Context, rules []index.Rule, id ksuid.KSUID) ([]*index.Object, error) {
	r, err := b.engine.Get(ctx, data.RowObjectPath(b.pool.DataPath, id))
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

func (b *Branch) Stats(ctx context.Context, snap commits.View) (info BranchStats, err error) {
	ch := make(chan data.Object)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = ScanSpan(ctx, snap, nil, b.pool.Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for object := range ch {
		info.Size += object.RowSize
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(object.First, object.Last, b.pool.Layout.Order)
		} else {
			poolSpan.Extend(object.First)
			poolSpan.Extend(object.Last)
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
