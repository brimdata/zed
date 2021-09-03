package commits

import (
	"errors"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
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

func (p *Patch) Segments() []ksuid.KSUID {
	var ids []ksuid.KSUID
	for _, segment := range p.diff.SelectAll() {
		ids = append(ids, segment.ID)
	}
	return ids
}

func (p *Patch) Adds() []ksuid.KSUID {
	var ids []ksuid.KSUID
	for _, segment := range p.base.SelectAll() {
		ids = append(ids, segment.ID)
	}
	return ids
}

func (p *Patch) Deletes() []ksuid.KSUID {
	return p.deletes
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

func (p *Patch) NewCommitObject(parent ksuid.KSUID, retries int, author, message string) *Object {
	o := NewObject(parent, author, message, retries)
	for _, id := range p.deletes {
		o.appendDelete(id)
	}
	for _, s := range p.diff.segments {
		o.appendAdd(s)
	}
	return o
}

//XXX We need to handle more than add/delete.  See issue #3000.
func (p *Patch) Undo(tip *Snapshot, commit, parent ksuid.KSUID, retries int, author, message string) (*Object, error) {
	object := NewObject(parent, author, message, retries)
	// For each segment in the patch that is also in the tip, we do a delete.
	segments := p.diff.SelectAll()
	for _, segment := range segments {
		if Exists(tip, segment.ID) {
			object.appendDelete(segment.ID)
		}
	}
	// For each delete in the patch that is not in the tip, we do an add.
	for _, id := range p.deletes {
		segment, err := tip.Lookup(id)
		if err == nil {
			object.appendAdd(segment)
		}
	}
	if len(object.Actions) == 1 {
		return nil, errors.New("undo commit is empty")
	}
	return object, nil
}

func (p *Patch) OverlappingDeletes(with *Patch) []ksuid.KSUID {
	lookup := make(map[ksuid.KSUID]struct{})
	for _, id := range p.deletes {
		lookup[id] = struct{}{}
	}
	var ids []ksuid.KSUID
	for _, id := range with.deletes {
		if _, ok := lookup[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}
