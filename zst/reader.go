package zst

import (
	"context"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng/resolver"
)

// Reader implements the zbuf.Reader and io.Closer.  It reads a columnar
// zst object to generate a stream of zng.Records.  It also has methods
// to read metainformation for test and debugging.
type Reader struct {
	*Object
	zbuf.Reader
}

// NewReader returns a Reader ready to read a zst object as zng.Records.
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

func NewReaderFromPath(ctx context.Context, zctx *resolver.Context, path string) (*Reader, error) {
	object, err := NewObjectFromPath(ctx, zctx, path)
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

func NewReaderFromSeeker(zctx *resolver.Context, seeker Seeker) (*Reader, error) {
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
