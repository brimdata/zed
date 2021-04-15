package commit

import (
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

// A snapshot summarizes the pool state at a given point in the journal.
// XXX TBD: sort log by span and stack snapshot as a base layer
// followed by change sets for each base-layer generation.  This is what
// we will store in the journal sub-pool when this gets implemented.
// See issue #XXX.
// Also, snapshots should have index updates so each segment in the
// snapshot could include which indexes are attached to it.
type Snapshot struct {
	at       journal.ID
	order    zbuf.Order
	segments []segment.Reference
}

func newSnapshot(at journal.ID, order zbuf.Order) *Snapshot {
	return &Snapshot{at: at, order: order}
}

func (s *Snapshot) Segments() []segment.Reference {
	return s.segments
}

func (s *Snapshot) AddSegment(seg segment.Reference) {
	s.segments = append(s.segments, seg)
}

func (s *Snapshot) DeleteSegment(id ksuid.KSUID) {
	panic("TBD")
}

func (s *Snapshot) Select(span nano.Span) []*segment.Reference {
	var segments []*segment.Reference
	for k, seg := range s.segments {
		if span.Overlaps(seg.Span()) {
			segments = append(segments, &s.segments[k])
		}
	}
	return segments
}
