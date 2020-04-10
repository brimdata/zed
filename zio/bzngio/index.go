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

type mark struct {
	Ts     nano.Ts
	Offset uint64
}

// XXX document me
func NewIndex() Index {
	return Index{}
}

// NewReader creates a new zbuf.Reader for reading the given indexed bzng
// file.  It tries to build and use an in-memory time index in order to
// handle requests with limited time spans efficiently.
func (i *Index) NewReader(f *os.File, zctx *resolver.Context, span nano.Span) (zbuf.ReadCloser, error) {
	if i.indexReady {
		return newRangeReader(f, zctx, i.index, span)
	} else {
		return newIndexReader(i, f, zctx), nil
	}
}

// indexReader is a zbuf.Reader that also builds an index as it reads.
type indexReader struct {
	*Reader
	closer io.Closer
	parent *Index
	marks  []mark
	ts     nano.Ts
}

func newIndexReader(parent *Index, reader io.ReadCloser, zctx *resolver.Context) *indexReader {
	return &indexReader{
		Reader: NewReader(reader, zctx),
		closer: reader,
		parent: parent,
	}
}

func (i *indexReader) Read() (*zng.Record, error) {
	rec, reset, err := i.Reader.ReadWithResets()
	if err != nil {
		return nil, err
	}

	if reset > 0 {
		ts := rec.Ts
		if ts > i.ts {
			i.ts = ts
			i.marks = append(i.marks, mark{ts, reset})
		}
	}

	if rec == nil {
		i.parent.index = i.marks
		i.parent.indexReady = true
	}
	return rec, nil
}

func (i *indexReader) Close() error {
	return i.closer.Close()
}

// rangeReader is a wrapper around bzngio.Reader that uses an in-memory
// index to reduce the I/O needed to get matching records when reading a
// large bzng file that includes reset markers and a nano.Span that refers
// to a smaller time range within the file.
type rangeReader struct {
	*Reader
	closer io.Closer
	end    nano.Ts
}

func newRangeReader(f *os.File, zctx *resolver.Context, index []mark, span nano.Span) (*rangeReader, error) {
	var off uint64
	// XXX binary search
	for _, mark := range index {
		if mark.Ts > span.Ts {
			break
		}
		off = mark.Offset
	}
	if off > 0 {
		newoff, err := f.Seek(int64(off), 0)
		if err != nil {
			return nil, err
		}
		if newoff != int64(off) {
			return nil, errors.New("file truncated") //XXX
		}
	}
	return &rangeReader{
		Reader: NewReader(f, zctx),
		closer: f,
		end:    span.End(),
	}, nil
}

func (r *rangeReader) Read() (*zng.Record, error) {
	rec, err := r.Reader.Read()
	if err != nil {
		return nil, err
	}
	if rec != nil && rec.Ts > r.end {
		rec = nil
	}
	return rec, nil
}

func (r *rangeReader) Close() error {
	return r.closer.Close()
}
