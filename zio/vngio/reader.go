package vngio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/vng"
)

func NewReader(zctx *zed.Context, r io.Reader) (*vng.Reader, error) {
	s, ok := r.(io.Seeker)
	if sr, ok2 := r.(storage.Reader); !ok || (ok2 && !storage.IsSeekable(sr)) {
		return nil, errors.New("VNG must be used with a seekable input")
	}
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, errors.New("VNG must be used with an io.ReaderAt")
	}
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	o, err := vng.NewObject(zctx, ra, size)
	if err != nil {
		return nil, err
	}
	return vng.NewReader(o)
}
