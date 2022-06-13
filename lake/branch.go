package lake

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	maxCommitRetries  = 10
	maxMessageObjects = 10
)

var (
	ErrCommitFailed      = fmt.Errorf("exceeded max update attempts (%d) to branch tip: commit failed", maxCommitRetries)
	ErrInvalidCommitMeta = errors.New("cannot parse ZSON string")
)

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

func (b *Branch) Load(ctx context.Context, zctx *zed.Context, r zio.Reader, author, message, meta string) (ksuid.KSUID, error) {
	w, err := NewWriter(ctx, zctx, b.pool)
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
	appMeta, err := loadMeta(zctx, meta)
	if err != nil {
		return ksuid.Nil, err
	}
	// The load operation has only added new objects so we know its
	// safe to merge at the tip and there can be no conflicts
	// with other concurrent writers (except for updating the branch pointer
	// which is handled by Branch.commit)
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		return commits.NewAddsObject(parent.Commit, retries, author, message, *appMeta, objects), nil
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

func loadMeta(zctx *zed.Context, meta string) (*zed.Value, error) {
	if meta == "" {
		return &zed.Value{zed.TypeNull, nil}, nil
	}
	zv, err := zson.ParseValue(zed.NewContext(), meta)
	if err != nil {
		return zctx.Missing(), fmt.Errorf("%w %s: %v", ErrInvalidCommitMeta, zv, err)
	}
	return zv, nil
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

func (b *Branch) DeleteByPredicate(ctx context.Context, c runtime.Compiler, src string, author, message, meta string) (ksuid.KSUID, error) {
	zctx := zed.NewContext()
	appMeta, err := loadMeta(zctx, meta)
	if err != nil {
		return ksuid.Nil, err
	}
	val, op, err := c.ParseRangeExpr(zctx, src, b.pool.Layout)
	if err != nil {
		return ksuid.Nil, err
	}
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		base, err := b.pool.commits.Snapshot(ctx, parent.Commit)
		if err != nil {
			return nil, err
		}
		copies, deletes, err := b.getDbpObjects(ctx, c, val, op, parent.Commit)
		if err != nil {
			return nil, err
		}
		count := len(copies) + len(deletes)
		if count == 0 {
			return nil, errors.New("nothing to delete")
		}
		// XXX Copied should be deleted if the commit fails.
		copied, err := b.dbpCopies(ctx, c, base, copies, src)
		if err != nil {
			return nil, err
		}
		patch := commits.NewPatch(base)
		for _, o := range copied {
			patch.AddDataObject(&o)
		}
		for _, id := range append(copies, deletes...) {
			patch.DeleteObject(id)
		}
		if message == "" {
			plural := "s"
			if count == 1 {
				plural = ""
			}
			message = fmt.Sprintf("deleted %d object%s using predicate %s", count, plural, src)
		}
		commit := patch.NewCommitObject(parent.Commit, retries, author, message, *appMeta)
		return commit, nil
	})
}

func (b *Branch) dbpCopies(ctx context.Context, c runtime.Compiler, snap *commits.Snapshot, copies []ksuid.KSUID, filter string) ([]data.Object, error) {
	if len(copies) == 0 {
		return nil, nil
	}
	zctx := zed.NewContext()
	readers := make([]zio.Reader, len(copies))
	// XXX instead of opening each object individually like this we should
	// instead create an ephemeral branch/pool that has only the copied
	// objects then run our delete query on this subset.
	for i, id := range copies {
		o, err := snap.Lookup(id)
		if err != nil {
			return nil, err
		}
		// XXX Ideally we'd be opening objects via data.NewReader so we can
		// leverage the seek index to skip over irrelavant segments but we don't
		// have a way to translate the delete filter into an extent.Span, so
		// just read the whole file.
		objectPath := o.RowObjectPath(b.pool.DataPath)
		reader, err := b.pool.engine.Get(ctx, objectPath)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		readers[i] = zngio.NewReader(reader, zctx)
		defer readers[i].(*zngio.Reader).Close()
	}
	// Keeps values that don't fit the filter by adding "not".
	flowgraph, err := c.Parse("not " + filter)
	if err != nil {
		return nil, err
	}
	r := zio.NewCombiner(ctx, readers)
	q, err := runtime.CompileQuery(ctx, zctx, c, flowgraph, []zio.Reader{r})
	if err != nil {
		return nil, err
	}
	defer q.Close()
	w, err := NewWriter(ctx, zctx, b.pool)
	if err != nil {
		return nil, err
	}
	err = zbuf.CopyPuller(w, q)
	if err2 := w.Close(); err == nil {
		err = err2
	}
	if err != nil {
		for _, o := range w.Objects() {
			o.Remove(ctx, b.pool.engine, b.pool.DataPath)
		}
		return nil, err
	}
	return w.Objects(), nil
}

// getDbpObjects gets the object IDs of objects effected by a delete by predicate
// operation.
func (b *Branch) getDbpObjects(ctx context.Context, c runtime.Compiler, val *zed.Value, op string, commit ksuid.KSUID) ([]ksuid.KSUID, []ksuid.KSUID, error) {
	const dbp = `
const THRESH = %s
from %s@%s:objects
| {
	id: id,
	lower: compare(meta.first, meta.last, %t) < 0 ? meta.first : meta.last,
	upper: compare(meta.first, meta.last, %t) < 0 ? meta.last : meta.first
  }
| switch (
  case %s %s THRESH => deletes:=collect(id)
  case %s %s THRESH => copies:=collect(id)
)`
	deletesField, copiesField := "upper", "lower"
	if op == ">=" || op == ">" {
		deletesField, copiesField = copiesField, deletesField
	}
	nullsMax := b.pool.Layout.Order == order.Asc
	src := fmt.Sprintf(dbp, zson.MustFormatValue(val), b.pool.ID, commit, nullsMax, nullsMax, deletesField, op, copiesField, op)
	flowgraph, err := c.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	q, err := runtime.CompileLakeQuery(ctx, zed.NewContext(), c, flowgraph, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	defer q.Close()
	var objects struct {
		Deletes []ksuid.KSUID `zed:"deletes"`
		Copies  []ksuid.KSUID `zed:"copies"`
	}
	r := q.AsReader()
	for {
		val, err := r.Read()
		if err != nil {
			return nil, nil, err
		}
		if val == nil {
			break
		}
		if err := zson.UnmarshalZNG(val, &objects); err != nil {
			return nil, nil, err
		}
	}
	return objects.Copies, objects.Deletes, nil
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
	return childPatch.NewCommitObject(parent.Commit, retries, author, message, zed.Value{zed.TypeNull, nil}), nil
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

func (b *Branch) ApplyIndexRules(ctx context.Context, c runtime.Compiler, rules []index.Rule, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	idxrefs := make([]*index.Object, 0, len(rules)*len(ids))
	for _, id := range ids {
		//XXX make issue for this.
		// This could be easily parallized with errgroup.
		refs, err := b.indexObject(ctx, c, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		idxrefs = append(idxrefs, refs...)
	}
	author := "indexer"
	message := indexMessage(rules)
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		return commits.NewAddIndexesObject(parent.Commit, author, message, retries, idxrefs), nil
	})
}

func (b *Branch) UpdateIndex(ctx context.Context, c runtime.Compiler, rules []index.Rule) (ksuid.KSUID, error) {
	snap, err := b.pool.commits.Snapshot(ctx, b.Commit)
	if err != nil {
		return ksuid.Nil, err
	}
	var objects []*index.Object
	for id, rules := range snap.Unindexed(rules) {
		o, err := b.indexObject(ctx, c, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		objects = append(objects, o...)
	}
	if len(objects) == 0 {
		return ksuid.Nil, commits.ErrEmptyTransaction
	}
	const author = "indexer"
	message := indexMessage(rules)
	return b.commit(ctx, func(parent *branches.Config, retries int) (*commits.Object, error) {
		return commits.NewAddIndexesObject(parent.Commit, author, message, retries, objects), nil
	})
}

func indexMessage(rules []index.Rule) string {
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

func (b *Branch) indexObject(ctx context.Context, c runtime.Compiler, rules []index.Rule, id ksuid.KSUID) ([]*index.Object, error) {
	r, err := b.engine.Get(ctx, data.RowObjectPath(b.pool.DataPath, id))
	if err != nil {
		return nil, err
	}
	reader := zngio.NewReader(r, zed.NewContext())
	defer reader.Close()
	w, err := index.NewCombiner(ctx, c, b.engine, b.pool.IndexPath, rules, id)
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
