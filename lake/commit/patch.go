package commit

import (
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

// A Patch represents a difference between a base snapshot and the patched
// snapshot.  Patch implements View so either a patch or a base snapshot
// can be traversed in the same manner.  Furthermore, patches can be easily
// chained to implement a sequence of patches to a base snapshot.
type Patch struct {
	base    *Snapshot
	diff    *Snapshot
	deletes []ksuid.KSUID
}

func NewPatch(base *Snapshot) *Patch {
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

func (p *Patch) Select(span nano.Span) Segments {
	segments := p.base.Select(span)
	segments.Append(p.diff.Select(span))
	return segments
}

func (p *Patch) AddSegment(seg *segment.Reference) error {
	if p.base.Exists(seg.ID) {
		return ErrExists
	}
	return p.diff.AddSegment(seg)
}

func (p *Patch) DeleteSegment(id ksuid.KSUID) error {
	if p.diff.Exists(id) {
		return p.diff.DeleteSegment(id)
	}
	if !p.base.Exists(id) {
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
