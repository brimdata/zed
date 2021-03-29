package zngio

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

type Ordering int

const (
	OrderUnknown Ordering = iota
	OrderAscending
	OrderDescending
	OrderUnsorted
)

type TimeIndex struct {
	mu         sync.Mutex
	order      Ordering
	index      []mark
	indexReady bool
}

type mark struct {
	Ts     nano.Ts
	Offset int64
}

// NewIndex creates a new Index object, which is the container that holds
// the in-memory index for a (b)zng file.  The first call to NewReader()
// will return a reader that scans the entire file, building a time-based
// index in the process, subsequent readers can use this index to read
// only the relevant zng streams from the underlying file.
func NewTimeIndex() *TimeIndex {
	return &TimeIndex{}
}

// Create a new reader for the given zng file.  Only records with timestamps
// that fall within the time range indicated by span will be emitted by
// the returned Reader object.
func (ti *TimeIndex) NewReader(f *os.File, zctx *resolver.Context, span nano.Span) (zbuf.ReadCloser, error) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if ti.indexReady {
		return newRangeReader(f, zctx, ti.order, ti.index, span)
	}

	return &indexReader{
		Reader: *NewReader(f, zctx),
		Closer: f,
		start:  span.Ts,
		end:    span.End(),
		parent: ti,
	}, nil
}

// indexReader is a zbuf.Reader that also builds an index as it reads.
type indexReader struct {
	Reader
	io.Closer
	start         nano.Ts
	end           nano.Ts
	parent        *TimeIndex
	order         Ordering
	marks         []mark
	lastSOS       int64
	lastTs        nano.Ts
	lastIndexedTs nano.Ts
}

func (i *indexReader) Read() (*zng.Record, error) {
	for {
		rec, err := i.readOne()
		if err != nil {
			return nil, err
		}

		if rec == nil {
			i.parent.mu.Lock()
			defer i.parent.mu.Unlock()
			i.parent.order = i.order
			i.parent.index = i.marks
			i.parent.indexReady = true
			return nil, nil
		}

		if rec.Ts() < i.start {
			continue
		}
		if rec.Ts() <= i.end {
			return rec, nil
		}

		// This record falls after the end of the requested time
		// span.  Spin through this loop until we hit EOF anyway
		// to finish building the index.
		// XXX this will be wasteful if small ranges near the
		// start of the file is all that is ever read.  revisit this...
	}
}

func (i *indexReader) readOne() (*zng.Record, error) {
	rec, err := i.Reader.Read()
	if err != nil || rec == nil {
		return nil, err
	}

	if i.lastTs != 0 {
		switch i.order {
		case OrderUnknown:
			if rec.Ts() > i.lastTs {
				i.order = OrderAscending
			} else if rec.Ts() < i.lastTs {
				i.order = OrderDescending
			}
		case OrderAscending:
			if rec.Ts() < i.lastTs {
				i.order = OrderUnsorted
			}
		case OrderDescending:
			if rec.Ts() > i.lastTs {
				i.order = OrderUnsorted
			}
		}
	}
	i.lastTs = rec.Ts()

	sos := i.Reader.LastSOS()
	if sos != i.lastSOS {
		i.lastSOS = sos
		ts := rec.Ts()
		if ts != i.lastIndexedTs {
			i.lastIndexedTs = ts
			i.marks = append(i.marks, mark{ts, sos})
		}
	}

	return rec, nil
}

// rangeReader is a wrapper around zngio.Reader that uses an in-memory
// index to reduce the I/O needed to get matching records when reading a
// large zng file that includes sub-streams and a nano.Span that refers
// to a smaller time range within the file.
type rangeReader struct {
	Reader
	io.Closer
	order Ordering
	start nano.Ts
	end   nano.Ts
	nread uint64
}

func newRangeReader(f *os.File, zctx *resolver.Context, order Ordering, index []mark, span nano.Span) (*rangeReader, error) {
	var off int64

	if order == OrderAscending || order == OrderDescending {
		// Find the stream within the zng file that holds the
		// start time.  For a large index this could be optimized
		// with a binary search.
		for _, mark := range index {
			if order == OrderAscending && mark.Ts > span.Ts {
				break
			}
			if order == OrderDescending && mark.Ts < span.End() {
				break
			}
			off = mark.Offset
		}
	}

	if off > 0 {
		newoff, err := f.Seek(off, io.SeekStart)
		if err != nil {
			return nil, err
		}
		if newoff != int64(off) {
			return nil, errors.New("file truncated") //XXX
		}
	}
	return &rangeReader{
		Reader: *NewReader(f, zctx),
		Closer: f,
		order:  order,
		start:  span.Ts,
		end:    span.End(),
	}, nil
}

func (r *rangeReader) Read() (*zng.Record, error) {
	for {
		rec, err := r.Reader.Read()
		if err != nil {
			return nil, err
		}
		r.nread++
		if rec != nil {
			switch r.order {
			case OrderAscending:
				if rec.Ts() < r.start {
					continue
				}
				if rec.Ts() > r.end {
					rec = nil
				}
			case OrderDescending:
				if rec.Ts() > r.end {
					continue
				}
				if rec.Ts() < r.start {
					rec = nil
				}
			}
		}
		return rec, nil
	}
}

// Used from tests
func (r *rangeReader) reads() uint64 {
	return r.nread
}
