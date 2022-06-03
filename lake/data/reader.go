package data

import (
	"context"
	"io"

	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/pkg/storage"
)

type Reader struct {
	io.Reader
	io.Closer
	TotalBytes int64
	ReadBytes  int64
}

// NewReader returns a Reader for this data object. If the object has a seek index
// and if the provided span skips part of the object, the seek index will be used to
// limit the reading window of the returned reader.
func (o *Object) NewReader(ctx context.Context, engine storage.Engine, path *storage.URI, rg seekindex.Range) (*Reader, error) {
	objectPath := o.RowObjectPath(path)
	reader, err := engine.Get(ctx, objectPath)
	if err != nil {
		return nil, err
	}
	sr := &Reader{
		Reader:     reader,
		Closer:     reader,
		TotalBytes: o.Size,
		ReadBytes:  o.Size, //XXX
	}
	sr.ReadBytes = rg.Size()
	sr.Reader, err = rg.Reader(reader)
	return sr, err
}
