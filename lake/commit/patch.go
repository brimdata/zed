package commit

import (
	"context"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zngbytes"
	"github.com/segmentio/ksuid"
)

// A Patch represents a difference between a base snapshot and the patched
// snapshot.  Patch implements View so either a patch or a base snapshot
// can be traversed in the same manner.  Furthermore, patches can be easily
// chained to implement a sequence of patches to a base snapshot.
type Patch struct {
	base    View
	diff    *Snapshot
	deletes []ksuid.KSUID
}

func NewPatch(base View) *Patch {
	return &Patch{
		base: base,
		diff: NewSnapshot(),
	}
}

func (p *Patch) Lookup(id ksuid.KSUID) (*segment.Reference, error) {
	if s, err := p.diff.Lookup(id); err == nil {
		return s, nil
	}
	return p.base.Lookup(id)
}

func (p *Patch) Select(span extent.Span, o order.Which) Segments {
	segments := p.base.Select(span, o)
	segments.Append(p.diff.Select(span, o))
	return segments
}

func (p *Patch) SelectAll() Segments {
	segments := p.base.SelectAll()
	segments.Append(p.diff.SelectAll())
	return segments
}

func (p *Patch) AddSegment(seg *segment.Reference) error {
	if Exists(p.base, seg.ID) {
		return ErrExists
	}
	return p.diff.AddSegment(seg)
}

func (p *Patch) DeleteSegment(id ksuid.KSUID) error {
	if p.diff.Exists(id) {
		return p.diff.DeleteSegment(id)
	}
	if !Exists(p.base, id) {
		return ErrNotFound
	}
	// Keep track of the deletions from the base so we can add the
	// needed delete Actions when building the transaction patch.
	p.deletes = append(p.deletes, id)
	return nil
}

func (p *Patch) NewTransaction() *Transaction {
	adds := p.diff.segments
	txn := newTransaction(ksuid.New(), len(adds)+len(p.deletes))
	for _, id := range p.deletes {
		txn.appendDelete(id)
	}
	for _, s := range adds {
		txn.appendAdd(s)
	}
	return txn
}

//XXX this should be unified with the other loops that do play
func (p *Patch) PlayLog(ctx context.Context, log *Log, at journal.ID) error {
	if at == journal.Nil {
		var err error
		at, err = log.journal.ReadHead(ctx)
		if err != nil {
			return err
		}
		if at == journal.Nil {
			// Empty log.  Do nothing since playing an empty log
			// onto a patch does nothing to the patch but it
			// should not fail.
			return nil
		}
	}
	r, err := log.Open(ctx, at, journal.Nil)
	if err != nil {
		return err
	}
	reader := zngbytes.NewDeserializer(r, actions.JournalTypes)
	for {
		entry, err := reader.Read()
		if err != nil {
			return err
		}
		if entry == nil {
			break
		}
		action, ok := entry.(actions.Interface)
		if !ok {
			return badEntry(entry)
		}
		PlayAction(p, action)
	}
	return nil
}
