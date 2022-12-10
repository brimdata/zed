package zstio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zst"
)

func NewReader(zctx *zed.Context, r io.Reader) (*zst.Reader, error) {
	reader, ok := r.(storage.Reader)
	if ok {
		if !storage.IsSeekable(reader) {
			return nil, errors.New("zst must be used with a seekable input")
		}
		size, err := storage.Size(reader)
		if err != nil {
			return nil, err
		}
		o, err := zst.NewObject(zctx, reader, size)
		if err != nil {
			return nil, err
		}
		return zst.NewReader(o)
	}
	// This can't be the zed system (which always using package storage)
	// so it must be a third party using he zst library.  We could assert
	// for io.ReaderAt and io.Seeker and use the seeker to get the size
	// and make a zst reader this way, but for now, we just say this is not supported.
	return nil, errors.New("zst does not yet support non-storage package implementations")
}
