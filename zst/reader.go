package zst

import (
	"context"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

// Reader implements zio.Reader and io.Closer.  It reads a columnar
// zst object to generate a stream of zed.Records.  It also has methods
// to read metainformation for test and debugging.
type Reader struct {
	*Object
	zio.Reader
}

// NewReader returns a Reader ready to read a zst object as zed.Records.
// Close() should be called when done.  This embeds a zst.Object.
func NewReader(object *Object) (*Reader, error) {
	assembler, err := NewAssembler(object.assembly, object.seeker)
	if err != nil {
		return nil, err
	}
	return &Reader{
		Object: object,
		Reader: assembler,
	}, nil

}

func NewReaderFromPath(ctx context.Context, zctx *zson.Context, engine storage.Engine, path string) (*Reader, error) {
	object, err := NewObjectFromPath(ctx, zctx, engine, path)
	if err != nil {
		return nil, err
	}
	reader, err := NewReader(object)
	if err != nil {
		object.Close()
		return nil, err
	}
	return reader, nil
}

func NewReaderFromSeeker(zctx *zson.Context, seeker *storage.Seeker) (*Reader, error) {
	object, err := NewObjectFromSeeker(zctx, seeker)
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
