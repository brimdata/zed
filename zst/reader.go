package zst

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zst/column"
)

// Reader implements zio.Reader and io.Closer.  It reads a columnar
// zst object to generate a stream of zed.Records.
type Reader struct {
	*Object
	root    *column.Int64Reader
	readers []typedReader
	builder zcode.Builder
}

var _ zio.Reader = (*Reader)(nil)

// NewReader returns a Reader ready to read a zst object as zed.Records.
// Close() should be called when done.  This embeds a zst.Object.
func NewReader(o *Object, seeker *storage.Seeker) (*Reader, error) {
	root := column.NewInt64Reader(o.root, seeker)
	readers := make([]typedReader, 0, len(o.maps))
	for _, m := range o.maps {
		r, err := column.NewReader(m, seeker)
		if err != nil {
			return nil, err
		}
		readers = append(readers, typedReader{typ: m.Type(o.zctx), reader: r})
	}
	return &Reader{
		Object:  o,
		root:    root,
		readers: readers,
	}, nil

}

func NewReaderFromPath(ctx context.Context, zctx *zed.Context, engine storage.Engine, path string) (*Reader, error) {
	reader, ok := engine.(storage.Reader)
	if !ok {
		return nil, errors.New("zst must be used with a seekable input")
	}
	seeker, err := storage.NewSeeker(reader)
	if err != nil {
		return nil, errors.New("zst must be used with a seekable input")
	}
	object, err := NewObjectFromPath(ctx, zctx, engine, path)
	if err != nil {
		return nil, err
	}
	if err != nil {
		object.Close()
		return nil, err
	}
	r, err := NewReader(object, seeker)
	if err != nil {
		object.Close()
		return nil, err
	}
	return r, nil
}

func NewReaderFromSeeker(zctx *zed.Context, seeker *storage.Seeker) (*Reader, error) {
	object, err := NewObjectFromSeeker(zctx, seeker)
	if err != nil {
		return nil, err
	}
	reader, err := NewReader(object, seeker)
	if err != nil {
		// don't close object as we didn't open the seeker
		return nil, err
	}
	return reader, nil
}

func (r *Reader) Read() (*zed.Value, error) {
	r.builder.Reset()
	typeNo, err := r.root.Read()
	if err == io.EOF {
		return nil, nil
	}
	if typeNo < 0 || int(typeNo) >= len(r.readers) {
		return nil, fmt.Errorf("system error: type number out of range in ZST root metadata")
	}
	tr := r.readers[typeNo]
	if err := tr.reader.Read(&r.builder); err != nil {
		return nil, err
	}
	return zed.NewValue(tr.typ, r.builder.Bytes().Body()), nil
}
