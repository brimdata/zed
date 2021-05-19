package segment

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
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
func (r *Reference) NewReader(ctx context.Context, engine storage.Engine, path *storage.URI, scanRange extent.Span, cmp expr.ValueCompareFn) (*Reader, error) {
	objectPath := r.RowObjectPath(path)
	reader, err := engine.Get(ctx, objectPath)
	if err != nil {
		return nil, err
	}
	sr := &Reader{
		Reader:     reader,
		Closer:     reader,
		TotalBytes: r.RowSize,
		ReadBytes:  r.RowSize, //XXX
	}
	// If a whole segment has nulls for the key values, just return the
	// whole-segment reader.  Eventually, we will store keyless rows some
	// other way, perhaps in a sub-pool.
	if r.First.Bytes == nil || r.Last.Bytes == nil {
		return sr, nil
	}
	segspan := extent.NewGeneric(r.First, r.Last, cmp)
	if !segspan.Crop(scanRange) {
		return nil, fmt.Errorf("segment reader: segment does not intersect provided span: %s (segment range %s) (scan range %s)", path, segspan, scanRange)
	}
	if bytes.Equal(r.First.Bytes, segspan.First().Bytes) && bytes.Equal(r.Last.Bytes, segspan.Last().Bytes) {
		return sr, nil
	}
	indexReader, err := engine.Get(ctx, r.SeekObjectPath(path))
	if err != nil {
		if zqe.IsNotFound(err) {
			return sr, nil
		}
		return nil, err
	}
	defer indexReader.Close()
	rg, err := seekindex.Lookup(zngio.NewReader(indexReader, zson.NewContext()), scanRange.First(), scanRange.Last(), cmp)
	if err != nil {
		if zqe.IsNotFound(err) {
			return sr, nil
		}
		return nil, err
	}
	rg = rg.TrimEnd(sr.TotalBytes)
	sr.ReadBytes = rg.Size()
	sr.Reader, err = rg.Reader(reader)
	return sr, err
}
