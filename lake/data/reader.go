package data

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
)

type ObjectScan struct {
	Object
	ScanRange seekindex.Range
}

func NewObjectScan(o Object) *ObjectScan {
	return &ObjectScan{Object: o}
}

type Reader struct {
	io.Reader
	io.Closer
	TotalBytes int64
	ReadBytes  int64
}

// NewReader returns a Reader for this data object. If the object has a seek index
// and if the provided span skips part of the object, the seek index will be used to
// limit the reading window of the returned reader.
func (o *ObjectScan) NewReader(ctx context.Context, engine storage.Engine, path *storage.URI, scanRange extent.Span, cmp expr.CompareFn) (*Reader, error) {
	objectPath := o.RowObjectPath(path)
	reader, err := engine.Get(ctx, objectPath)
	if err != nil {
		return nil, err
	}
	span := extent.NewGeneric(o.First, o.Last, cmp)
	if !span.Crop(scanRange) {
		return nil, fmt.Errorf("data object reader: object does not intersect provided span: %s (object range %s) (scan range %s)", path, span, scanRange)
	}
	sr := &Reader{
		Reader:     reader,
		Closer:     reader,
		TotalBytes: o.Size,
		ReadBytes:  o.Size, //XXX
	}
	rg := seekindex.Range{0, math.MaxInt64}
	if !bytes.Equal(o.First.Bytes, span.First().Bytes) || !bytes.Equal(o.Last.Bytes, span.Last().Bytes) {
		indexReader, err := engine.Get(ctx, o.SeekObjectPath(path))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return sr, nil
			}
			return nil, err
		}
		defer indexReader.Close()
		rg, err = seekindex.Lookup(zngio.NewReader(indexReader, zed.NewContext()), scanRange.First(), scanRange.Last(), cmp)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return sr, nil
			}
			return nil, err
		}
	}
	if !o.ScanRange.IsZero() {
		rg = rg.Crop(o.ScanRange)
	}
	rg = rg.TrimEnd(sr.TotalBytes)
	sr.ReadBytes = rg.Size()
	sr.Reader, err = rg.Reader(reader)
	return sr, err
}
