package segment

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zqe"
)

type Reader struct {
	io.Reader
	io.Closer
	TotalBytes int64
	ReadBytes  int64
}

// NewReader returns a Reader for this segment. If the segment has a seek index and
// if the provided span skips part of the segment, the seek index will be used to
// limit the reading window of the returned reader.
func (r *Reference) NewReader(ctx context.Context, path iosrc.URI, readspan nano.Span) (*Reader, error) {
	segspan := r.Span()
	span := segspan.Intersect(readspan)
	objectPath := r.RowObjectPath(path)
	if span.Dur == 0 {
		return nil, fmt.Errorf("segment reader: segment does not intersect provided span: %s chunkspan %v readspan %v", path, segspan, readspan)
	}
	reader, err := iosrc.NewReader(ctx, objectPath)
	if err != nil {
		return nil, err
	}
	sr := &Reader{
		Reader:     reader,
		Closer:     reader,
		TotalBytes: r.Size,
		ReadBytes:  r.Size,
	}
	if span == segspan {
		return sr, nil
	}
	s, err := seekindex.Open(ctx, r.SeekObjectPath(path))
	if err != nil {
		if zqe.IsNotFound(err) {
			return sr, nil
		}
		return nil, err
	}
	defer s.Close()
	rg, err := s.Lookup(ctx, span)
	if err != nil {
		return nil, err
	}
	rg = rg.TrimEnd(sr.TotalBytes)
	sr.ReadBytes = rg.Size()
	sr.Reader, err = rg.LimitReader(reader)
	return sr, err
}
