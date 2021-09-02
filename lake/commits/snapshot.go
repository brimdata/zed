package commits

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/segmentio/ksuid"
)

var (
	ErrWriteConflict = errors.New("write conflict")
	ErrNotInCommit   = errors.New("data object not found in commit object")
)

type View interface {
	Lookup(ksuid.KSUID) (*segment.Reference, error)
	Select(extent.Span, order.Which) Segments
	SelectAll() Segments
}

type Writeable interface {
	View
	AddSegment(seg *segment.Reference) error
	DeleteSegment(id ksuid.KSUID) error
}

// A snapshot summarizes the pool state at any point in
// the commit object tree.
// XXX redefine snapshot as type map instead of struct
type Snapshot struct {
	segments map[ksuid.KSUID]*segment.Reference
}

func NewSnapshot() *Snapshot {
	return &Snapshot{
		segments: make(map[ksuid.KSUID]*segment.Reference),
	}
}

func (s *Snapshot) AddSegment(seg *segment.Reference) error {
	id := seg.ID
	if _, ok := s.segments[id]; ok {
		return fmt.Errorf("%s: add of a duplicate data object: %w", id, ErrWriteConflict)
	}
	s.segments[id] = seg
	return nil
}

func (s *Snapshot) DeleteSegment(id ksuid.KSUID) error {
	if _, ok := s.segments[id]; !ok {
		return fmt.Errorf("%s: delete of a non-existent data object: %w", id, ErrWriteConflict)
	}
	delete(s.segments, id)
	return nil
}

func Exists(view View, id ksuid.KSUID) bool {
	_, err := view.Lookup(id)
	return err == nil
}

func (s *Snapshot) Exists(id ksuid.KSUID) bool {
	return Exists(s, id)
}

func (s *Snapshot) Lookup(id ksuid.KSUID) (*segment.Reference, error) {
	seg, ok := s.segments[id]
	if !ok {
		return nil, fmt.Errorf("%s: %w", id, ErrNotFound)
	}
	return seg, nil
}

func (s *Snapshot) Select(scan extent.Span, o order.Which) Segments {
	var segments Segments
	for _, seg := range s.segments {
		segspan := seg.Span(o)
		if scan == nil || segspan == nil || extent.Overlaps(scan, segspan) {
			segments = append(segments, seg)
		}
	}
	return segments
}

func (s *Snapshot) SelectAll() Segments {
	var segments Segments
	for _, seg := range s.segments {
		segments = append(segments, seg)
	}
	return segments
}

func (s *Snapshot) Copy() *Snapshot {
	out := NewSnapshot()
	for key, val := range s.segments {
		out.segments[key] = val
	}
	return out
}

type Segments []*segment.Reference

func (s *Segments) Append(segments Segments) {
	*s = append(*s, segments...)
}

func PlayAction(w Writeable, action Action) error {
	if _, ok := action.(Action); !ok {
		return badObject(action)
	}
	//XXX other cases like actions.AddIndex etc coming soon...
	switch action := action.(type) {
	case *Add:
		w.AddSegment(&action.Segment)
	case *Delete:
		w.DeleteSegment(action.ID)
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
