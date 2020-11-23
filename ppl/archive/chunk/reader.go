package chunk

import (
	"context"
	"errors"
	"io"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/archive/seekindex"
	"github.com/brimsec/zq/zqe"
)

type Reader struct {
	io.Reader
	io.Closer
	TotalSize int64
	ReadSize  int64
}

// NewReader returns a Reader for this chunk. If the chunk has a seek index and
// if the provided span skips part of the chunk, the seek index will be used to
// limit the reading window of the returned reader.
func NewReader(ctx context.Context, chunk Chunk, span nano.Span) (*Reader, error) {
	cspan := chunk.Span()
	span = cspan.Intersect(span)
	if span.Dur == 0 {
		return nil, errors.New("chunk span does intersect provided span")
	}
	r, err := iosrc.NewReader(ctx, chunk.Path())
	if err != nil {
		return nil, err
	}
	cr := &Reader{
		Reader:    r,
		Closer:    r,
		TotalSize: chunk.Size,
		ReadSize:  chunk.Size,
	}
	if span == cspan {
		return cr, nil
	}
	s, err := seekindex.Open(ctx, chunk.SeekIndexPath())
	if err != nil {
		if zqe.IsNotFound(err) {
			return cr, nil
		}
		return nil, err
	}
	defer s.Close()
	rg, err := s.Lookup(ctx, span)
	if err != nil {
		return nil, err
	}
	rg = rg.TrimEnd(cr.TotalSize)
	cr.ReadSize = rg.Size()
	cr.Reader, err = rg.LimitReader(r)
	return cr, err
}
