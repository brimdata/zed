package bzngio

import (
	"errors"
	"io"
	"os"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Index struct {
	index      []mark
	indexReady bool
}

type IndexReader interface {
	zbuf.ReadCloser
	// count of how many records were read from disk, currently just
	// used for testing.
	Reads() uint64
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
func NewIndex() Index {
	return Index{}
}

func (i *Index) NewReader(f *os.File, zctx *resolver.Context, span nano.Span) (IndexReader, error) {
	if i.indexReady {
		return newRangeReader(f, zctx, i.index, span)
	} else {
		return &indexReader{
			Reader: *NewReader(f, zctx),
			closer: f,
			start:  span.Ts,
			end:    span.End(),
			parent: i,
		}, nil
	}
}

// indexReader is a zbuf.Reader that also builds an index as it reads.
type indexReader struct {
	Reader
	closer  io.Closer
	start   nano.Ts
	end     nano.Ts
	parent  *Index
	marks   []mark
	lastSOS int64
	lastTs  nano.Ts
}

func (i *indexReader) Read() (*zng.Record, error) {
	for {
		rec, err := i.readOne()
		if rec == nil {
			i.parent.index = i.marks
			i.parent.indexReady = true
			return nil, nil
		}

		if err != nil {
			return nil, err
		}

		if rec.Ts < i.start {
			continue
		}
		if rec.Ts <= i.end {
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

	sos := i.Reader.LastSOS()
	if sos != i.lastSOS {
		i.lastSOS = sos
		ts := rec.Ts
		if ts > i.lastTs {
			i.lastTs = ts
			i.marks = append(i.marks, mark{ts, sos})
		}
	}

	return rec, nil
}

func (i *indexReader) Close() error {
	return i.closer.Close()
}

// this is only used for testing and only called on rangeReader
func (i *indexReader) Reads() uint64 {
	return 0
}

// rangeReader is a wrapper around bzngio.Reader that uses an in-memory
// index to reduce the I/O needed to get matching records when reading a
// large bzng file that includes sub-streams and a nano.Span that refers
// to a smaller time range within the file.
type rangeReader struct {
	Reader
	closer io.Closer
	start  nano.Ts
	end    nano.Ts
	nread  uint64
}

func newRangeReader(f *os.File, zctx *resolver.Context, index []mark, span nano.Span) (*rangeReader, error) {
	var off int64
	// XXX binary search
	for _, mark := range index {
		if mark.Ts > span.Ts {
			break
		}
		off = mark.Offset
	}
	if off > 0 {
		newoff, err := f.Seek(off, 0)
		if err != nil {
			return nil, err
		}
		if newoff != int64(off) {
			return nil, errors.New("file truncated") //XXX
		}
	}
	return &rangeReader{
		Reader: *NewReader(f, zctx),
		closer: f,
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
		if rec.Ts < r.start {
			continue
		}
		if rec != nil && rec.Ts > r.end {
			rec = nil
		}
		return rec, nil
	}
}

func (r *rangeReader) Close() error {
	return r.closer.Close()
}

func (r *rangeReader) Reads() uint64 {
	return r.nread
}
