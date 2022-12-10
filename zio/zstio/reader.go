package zstio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zst"
)

func NewReader(zctx *zed.Context, r io.Reader) (*zst.Reader, error) {
	s, ok := r.(io.Seeker)
	if sr, ok2 := r.(storage.Reader); !ok || (ok2 && !storage.IsSeekable(sr)) {
		return nil, errors.New("zst must be used with a seekable input")
	}
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, errors.New("zst must be used with an io.ReaderAt")
	}
	size, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	o, err := zst.NewObject(zctx, ra, size)
	if err != nil {
		return nil, err
	}
	return zst.NewReader(o)
}
