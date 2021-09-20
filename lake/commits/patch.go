package commits

import (
	"errors"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/segmentio/ksuid"
)

// A Patch represents a difference between a base snapshot and the patched
// snapshot.  Patch implements View so either a patch or a base snapshot
// can be traversed in the same manner.  Furthermore, patches can be easily
// chained to implement a sequence of patches to a base snapshot.
type Patch struct {
	base           View
	diff           *Snapshot
	deletedObjects []ksuid.KSUID
	deletedIndexes []indexRef
}

var _ View = (*Patch)(nil)
var _ Writeable = (*Patch)(nil)

func NewPatch(base View) *Patch {
	return &Patch{
		base: base,
		diff: NewSnapshot(),
	}
}

func (p *Patch) Lookup(id ksuid.KSUID) (*data.Object, error) {
	if s, err := p.diff.Lookup(id); err == nil {
		return s, nil
	}
	return p.base.Lookup(id)
}

func (p *Patch) LookupIndex(ruleID, id ksuid.KSUID) (*index.Object, error) {
	if s, err := p.diff.LookupIndex(ruleID, id); err == nil {
		return s, nil
	}
	return p.base.LookupIndex(ruleID, id)
}

func (p *Patch) Select(span extent.Span, o order.Which) DataObjects {
	objects := p.base.Select(span, o)
	objects.Append(p.diff.Select(span, o))
	return objects
}

func (p *Patch) SelectAll() DataObjects {
	objects := p.base.SelectAll()
	objects.Append(p.diff.SelectAll())
	return objects
}

func (p *Patch) SelectIndexes(span extent.Span, o order.Which) []*index.Object {
	objects := p.base.SelectIndexes(span, o)
	return append(objects, p.diff.SelectIndexes(span, o)...)
}

func (p *Patch) DataObjects() []ksuid.KSUID {
	var ids []ksuid.KSUID
	for _, dataObject := range p.diff.SelectAll() {
		ids = append(ids, dataObject.ID)
	}
	return ids
}

func (p *Patch) AddDataObject(object *data.Object) error {
	if Exists(p.base, object.ID) {
		return ErrExists
	}
	return p.diff.AddDataObject(object)
}

func (p *Patch) DeleteObject(id ksuid.KSUID) error {
	if p.diff.Exists(id) {
		return p.diff.DeleteObject(id)
	}
	if !Exists(p.base, id) {
		return ErrNotFound
	}
	// Keep track of the deletions from the base so we can add the
	// needed delete Actions when building the transaction patch.
	p.deletedObjects = append(p.deletedObjects, id)
	return nil
}

func (p *Patch) AddIndexObject(object *index.Object) error {
	if IndexExists(p.base, object.Rule.RuleID(), object.ID) {
		return ErrExists
	}
	return p.diff.AddIndexObject(object)
}

type indexRef struct {
	id     ksuid.KSUID
	ruleID ksuid.KSUID
}

func (p *Patch) DeleteIndexObject(ruleID ksuid.KSUID, id ksuid.KSUID) error {
	if IndexExists(p.diff, ruleID, id) {
		return p.diff.DeleteIndexObject(ruleID, id)
	}
	if !IndexExists(p.base, ruleID, id) {
		return ErrNotFound
	}
	// Keep track of the deletions from the base so we can add the
	// needed delete Actions when building the transaction patch.
	p.deletedIndexes = append(p.deletedIndexes, indexRef{id, ruleID})
	return nil
}

func (p *Patch) NewCommitObject(parent ksuid.KSUID, retries int, author, message string) *Object {
	o := NewObject(parent, author, message, retries)
	for _, id := range p.deletedObjects {
		o.appendDelete(id)
	}
	for _, s := range p.diff.objects {
		o.appendAdd(s)
	}
	for _, r := range p.deletedIndexes {
		o.appendDeleteIndex(r.ruleID, r.id)
	}
	for _, s := range p.diff.indexes.All() {
		o.appendAddIndex(s)
	}
	return o
}

func (p *Patch) Revert(tip *Snapshot, commit, parent ksuid.KSUID, retries int, author, message string) (*Object, error) {
	object := NewObject(parent, author, message, retries)
	// For each data object in the patch that is also in the tip, we do a delete.
	for _, dataObject := range p.diff.SelectAll() {
		if Exists(tip, dataObject.ID) {
			object.appendDelete(dataObject.ID)
		}
	}
	// For each delete in the patch that is also deleted in the tip, we do an add.
	for _, id := range p.deletedObjects {
		dataObject, _ := tip.LookupDeleted(id)
		if dataObject != nil {
			object.appendAdd(dataObject)
		}
	}
	// For each index object in the patch that is also in the tip, we do a delete.
	for _, indexObject := range p.diff.indexes.All() {
		if tip.indexes.Exists(indexObject) {
			object.appendDeleteIndex(indexObject.Rule.RuleID(), indexObject.ID)
		}
	}
	// For each deleted index object in the patch that is also deleted in the
	// tip, we do an add.
	for _, ref := range p.deletedIndexes {
		if o := tip.deletedIndexes.Lookup(ref.ruleID, ref.id); o != nil {
			object.appendAddIndex(o)
		}
	}
	if len(object.Actions) == 1 {
		return nil, errors.New("revert commit is empty")
	}
	return object, nil
}

func (p *Patch) OverlappingDeletes(with *Patch) []ksuid.KSUID {
	lookup := make(map[ksuid.KSUID]struct{})
	for _, id := range p.deletedObjects {
		lookup[id] = struct{}{}
	}
	var ids []ksuid.KSUID
	for _, id := range with.deletedObjects {
		if _, ok := lookup[id]; ok {
			ids = append(ids, id)
		}
	}
	return ids
}
