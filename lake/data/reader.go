package data

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
func (o *Object) NewReader(ctx context.Context, engine storage.Engine, path *storage.URI, scanRange extent.Span, cmp expr.ValueCompareFn) (*Reader, error) {
	objectPath := o.RowObjectPath(path)
	reader, err := engine.Get(ctx, objectPath)
	if err != nil {
		return nil, err
	}
	sr := &Reader{
		Reader:     reader,
		Closer:     reader,
		TotalBytes: o.RowSize,
		ReadBytes:  o.RowSize, //XXX
	}
	// If a whole segment has nulls for the key values, just return the
	// whole-segment reader.  Eventually, we will store keyless rows some
	// other way, perhaps in a sub-pool.
	if o.First.Bytes == nil || o.Last.Bytes == nil {
		return sr, nil
	}
	span := extent.NewGeneric(o.First, o.Last, cmp)
	if !span.Crop(scanRange) {
		return nil, fmt.Errorf("segment reader: segment does not intersect provided span: %s (segment range %s) (scan range %s)", path, span, scanRange)
	}
	if bytes.Equal(o.First.Bytes, span.First().Bytes) && bytes.Equal(o.Last.Bytes, span.Last().Bytes) {
		return sr, nil
	}
	indexReader, err := engine.Get(ctx, o.SeekObjectPath(path))
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
