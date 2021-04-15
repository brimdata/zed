package commit

import (
	"errors"

	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

var (
	ErrExists   = errors.New("segment exists")
	ErrNotFound = errors.New("segment not found")
)

type View interface {
	Lookup(id ksuid.KSUID) (*segment.Reference, error)
	Select(span nano.Span) Segments
}

type Writeable interface {
	View
	AddSegment(seg *segment.Reference) error
	DeleteSegment(id ksuid.KSUID) error
}

// A snapshot summarizes the pool state at a given point in the journal.
type Snapshot struct {
	at       journal.ID
	segments map[ksuid.KSUID]*segment.Reference
}

func NewSnapshot() *Snapshot {
	return &Snapshot{
		segments: make(map[ksuid.KSUID]*segment.Reference),
	}
}

func newSnapshotAt(at journal.ID) *Snapshot {
	s := NewSnapshot()
	s.at = at
	return s
}

func (s *Snapshot) AddSegment(seg *segment.Reference) error {
	id := seg.ID
	if _, ok := s.segments[id]; ok {
		return ErrExists
	}
	s.segments[id] = seg
	return nil
}

func (s *Snapshot) DeleteSegment(id ksuid.KSUID) error {
	if _, ok := s.segments[id]; !ok {
		return ErrNotFound
	}
	delete(s.segments, id)
	return nil
}

func (s *Snapshot) Exists(id ksuid.KSUID) bool {
	_, ok := s.segments[id]
	return ok
}

func (s *Snapshot) Lookup(id ksuid.KSUID) (*segment.Reference, error) {
	seg, ok := s.segments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return seg, nil
}

func (s *Snapshot) Select(span nano.Span) Segments {
	var segments Segments
	for _, seg := range s.segments {
		if span.Overlaps(seg.Span()) {
			segments = append(segments, seg)
		}
	}
	return segments
}

type Segments []*segment.Reference

func (s *Segments) Append(segments Segments) {
	*s = append(*s, segments...)
}

func PlayAction(w Writeable, action actions.Interface) error {
	//XXX other cases like actions.AddIndex etc coming soon...
	switch action := action.(type) {
	case *actions.Add:
		w.AddSegment(&action.Segment)
	case *actions.Delete:
		w.DeleteSegment(action.ID)
	}
	return nil
}

// Play "plays" a recorded transaction into a whiteable snapshot.
func Play(w Writeable, txn *Transaction) error {
	for _, a := range txn.Actions {
		if err := PlayAction(w, a); err != nil {
			return err
		}
	}
	return nil
}
