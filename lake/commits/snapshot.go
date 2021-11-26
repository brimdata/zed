package commits

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var (
	ErrWriteConflict = errors.New("write conflict")
	ErrNotInCommit   = errors.New("data object not found in commit object")
)

type View interface {
	Lookup(ksuid.KSUID) (*data.Object, error)
	LookupIndex(ksuid.KSUID, ksuid.KSUID) (*index.Object, error)
	LookupIndexObjectRules(ksuid.KSUID) ([]index.Rule, error)
	Select(extent.Span, order.Which) DataObjects
	SelectAll() DataObjects
	SelectIndexes(extent.Span, order.Which) []*index.Object
}

type Writeable interface {
	View
	AddDataObject(*data.Object) error
	DeleteObject(ksuid.KSUID) error
	AddIndexObject(*index.Object) error
	DeleteIndexObject(ksuid.KSUID, ksuid.KSUID) error
}

// A snapshot summarizes the pool state at any point in
// the commit object tree.
// XXX redefine snapshot as type map instead of struct
type Snapshot struct {
	objects        map[ksuid.KSUID]*data.Object
	deletedObjects map[ksuid.KSUID]*data.Object
	indexes        index.Map
	deletedIndexes index.Map
}

var _ View = (*Snapshot)(nil)
var _ Writeable = (*Snapshot)(nil)

func NewSnapshot() *Snapshot {
	return &Snapshot{
		objects:        make(map[ksuid.KSUID]*data.Object),
		deletedObjects: make(map[ksuid.KSUID]*data.Object),
		indexes:        make(index.Map),
		deletedIndexes: make(index.Map),
	}
}

func (s *Snapshot) AddDataObject(object *data.Object) error {
	id := object.ID
	if _, ok := s.objects[id]; ok {
		return fmt.Errorf("%s: add of a duplicate data object: %w", id, ErrWriteConflict)
	}
	s.objects[id] = object
	delete(s.deletedObjects, id)
	return nil
}

func (s *Snapshot) DeleteObject(id ksuid.KSUID) error {
	object, ok := s.objects[id]
	if !ok {
		return fmt.Errorf("%s: delete of a non-existent data object: %w", id, ErrWriteConflict)
	}
	delete(s.objects, id)
	s.deletedObjects[id] = object
	return nil
}

func (s *Snapshot) AddIndexObject(object *index.Object) error {
	id := object.ID
	if s.indexes.Lookup(object.Rule.RuleID(), id) != nil {
		return fmt.Errorf("%s: add of a duplicate index object: %w", id, ErrWriteConflict)
	}
	s.indexes.Insert(object)
	s.deletedIndexes.Delete(object.Rule.RuleID(), id)
	return nil
}

func (s *Snapshot) DeleteIndexObject(ruleID ksuid.KSUID, id ksuid.KSUID) error {
	object := s.indexes.Lookup(ruleID, id)
	if object == nil {
		return fmt.Errorf("%s: delete of a non-existent index object: %w", index.ObjectName(ruleID, id), ErrWriteConflict)
	}
	s.indexes.Delete(ruleID, id)
	s.deletedIndexes.Insert(object)
	return nil
}

func Exists(view View, id ksuid.KSUID) bool {
	_, err := view.Lookup(id)
	return err == nil
}

func (s *Snapshot) Exists(id ksuid.KSUID) bool {
	return Exists(s, id)
}

func (s *Snapshot) Lookup(id ksuid.KSUID) (*data.Object, error) {
	o, ok := s.objects[id]
	if !ok {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	return o, nil
}

func IndexExists(view View, ruleID, id ksuid.KSUID) bool {
	_, err := view.LookupIndex(ruleID, id)
	return err == nil
}

func (s *Snapshot) LookupIndex(ruleID, id ksuid.KSUID) (*index.Object, error) {
	if o := s.indexes.Lookup(ruleID, id); o != nil {
		return o, nil
	}
	return nil, fmt.Errorf("%s: %w", index.ObjectName(ruleID, id), ErrNotFound)
}

func (s *Snapshot) LookupIndexObjectRules(id ksuid.KSUID) ([]index.Rule, error) {
	r, ok := s.indexes[id]
	if !ok {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	return r.Rules(), nil
}

func (s *Snapshot) LookupDeleted(id ksuid.KSUID) (*data.Object, error) {
	o, ok := s.deletedObjects[id]
	if !ok {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	return o, nil
}

func (s *Snapshot) Select(scan extent.Span, order order.Which) DataObjects {
	var objects DataObjects
	for _, o := range s.objects {
		segspan := o.Span(order)
		if scan == nil || segspan == nil || extent.Overlaps(scan, segspan) {
			objects = append(objects, o)
		}
	}
	return objects
}

func (s *Snapshot) SelectAll() DataObjects {
	var objects DataObjects
	for _, o := range s.objects {
		objects = append(objects, o)
	}
	return objects
}

func (s *Snapshot) SelectIndexes(scan extent.Span, order order.Which) []*index.Object {
	var indexes []*index.Object
	for _, i := range s.indexes.All() {
		o, ok := s.objects[i.ID]
		if !ok {
			continue
		}
		segspan := o.Span(order)
		if scan == nil || segspan == nil || extent.Overlaps(scan, segspan) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (s *Snapshot) Unindexed(rules []index.Rule) map[ksuid.KSUID][]index.Rule {
	unindexed := make(map[ksuid.KSUID][]index.Rule)
	for id := range s.objects {
		to := rules
		if o, ok := s.indexes[id]; ok {
			to = o.Missing(rules)
		}
		if len(to) > 0 {
			unindexed[id] = to
		}
	}
	return unindexed
}

func (s *Snapshot) Copy() *Snapshot {
	out := NewSnapshot()
	for key, val := range s.objects {
		out.objects[key] = val
	}
	for key, val := range s.deletedObjects {
		out.deletedObjects[key] = val
	}
	out.indexes = s.indexes.Copy()
	out.deletedIndexes = s.deletedIndexes.Copy()
	return out
}

// serialize serializes a snapshot as a sequence of actions.  Commit IDs are
// omitted from actions since they are neither available here nor required
// during deserialization.  Deleted entities are serialized as an add-delete
// sequence to meet the requirements of DeleteObject and DeleteIndexObject.
func (s *Snapshot) serialize() ([]byte, error) {
	zs := zngbytes.NewSerializer()
	zs.Decorate(zson.StylePackage)
	for _, o := range s.objects {
		if err := zs.Write(&Add{Object: *o}); err != nil {
			return nil, err
		}
	}
	for _, o := range s.deletedObjects {
		if err := zs.Write(&Add{Object: *o}); err != nil {
			return nil, err
		}
		if err := zs.Write(&Delete{ID: o.ID}); err != nil {
			return nil, err
		}
	}
	for _, objectRule := range s.indexes {
		for _, o := range objectRule {
			if err := zs.Write(&AddIndex{Object: *o}); err != nil {
				return nil, err
			}
		}
	}
	for _, objectRule := range s.deletedIndexes {
		for _, o := range objectRule {
			if err := zs.Write(&AddIndex{Object: *o}); err != nil {
				return nil, err
			}
			if err := zs.Write(&DeleteIndex{ID: o.ID, RuleID: o.Rule.RuleID()}); err != nil {
				return nil, err
			}
		}
	}
	if err := zs.Close(); err != nil {
		return nil, err
	}
	return zs.Bytes(), nil
}

func decodeSnapshot(r io.Reader) (*Snapshot, error) {
	s := NewSnapshot()
	zd := zngbytes.NewDeserializer(r, ActionTypes)
	for {
		entry, err := zd.Read()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			return s, nil
		}
		action, ok := entry.(Action)
		if !ok {
			return nil, fmt.Errorf("internal error: corrupt snapshot contains unknown entry type %T", entry)
		}
		if err := PlayAction(s, action); err != nil {
			return nil, err
		}
	}
}

type DataObjects []*data.Object

func (d *DataObjects) Append(objects DataObjects) {
	*d = append(*d, objects...)
}

func PlayAction(w Writeable, action Action) error {
	if _, ok := action.(Action); !ok {
		return badObject(action)
	}
	switch action := action.(type) {
	case *Add:
		w.AddDataObject(&action.Object)
	case *Delete:
		w.DeleteObject(action.ID)
	case *AddIndex:
		w.AddIndexObject(&action.Object)
	case *DeleteIndex:
		w.DeleteIndexObject(action.RuleID, action.ID)
	}
	return nil
}

// Play "plays" a recorded transaction into a writeable snapshot.
func Play(w Writeable, o *Object) error {
	for _, a := range o.Actions {
		if err := PlayAction(w, a); err != nil {
			return err
		}
	}
	return nil
}
