package vngio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/vng"
)

func NewReader(zctx *zed.Context, r io.Reader) (*vng.Reader, error) {
	reader, ok := r.(storage.Reader)
	if ok {
		if !storage.IsSeekable(reader) {
			return nil, errors.New("VNG must be used with a seekable input")
		}
		size, err := storage.Size(reader)
		if err != nil {
			return nil, err
		}
		o, err := vng.NewObject(zctx, reader, size)
		if err != nil {
			return nil, err
		}
		return vng.NewReader(o)
	}
	// This can't be the zed system (which always using package storage)
	// so it must be a third party using he VNG library.  We could assert
	// for io.ReaderAt and io.Seeker and use the seeker to get the size
	// and make a VNG reader this way, but for now, we just say this is not supported.
	return nil, errors.New("VNG does not yet support non-storage package implementations")
}
