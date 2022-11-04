package zst

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zst/vector"
)

// Reader implements zio.Reader and io.Closer.  It reads a vector
// zst object to generate a stream of zed.Records.
type Reader struct {
	*Object
	root    *vector.Int64Reader
	readers []typedReader
	builder zcode.Builder
	val     zed.Value
}

type typedReader struct {
	typ    zed.Type
	reader vector.Reader
}

var _ zio.Reader = (*Reader)(nil)

// NewReader returns a Reader ready to read a zst object as zed.Records.
// Close() should be called when done.  This embeds a zst.Object.
func NewReader(o *Object) (*Reader, error) {
	root := vector.NewInt64Reader(o.root, o.readerAt)
	readers := make([]typedReader, 0, len(o.maps))
	for _, m := range o.maps {
		r, err := vector.NewReader(m, o.readerAt)
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
	object, err := NewObjectFromPath(ctx, zctx, engine, path)
	if err != nil {
		return nil, err
	}
	if err != nil {
		object.Close()
		return nil, err
	}
	r, err := NewReader(object)
	if err != nil {
		object.Close()
		return nil, err
	}
	return r, nil
}

func NewReaderFromStorageReader(zctx *zed.Context, r storage.Reader) (*Reader, error) {
	object, err := NewObjectFromStorageReaderNoCloser(zctx, r)
	if err != nil {
		return nil, err
	}
	reader, err := NewReader(object)
	if err != nil {
		// don't close object as we didn't open the seeker
		return nil, err
	}
	return reader, nil
}

func (r *Reader) Read() (*zed.Value, error) {
	r.builder.Truncate()
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
	r.val = *zed.NewValue(tr.typ, r.builder.Bytes().Body())
	return &r.val, nil
}
