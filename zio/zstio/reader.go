package zstio

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zst"
)

func NewReader(r io.Reader, zctx *zed.Context) (*zst.Reader, error) {
	reader, ok := r.(storage.Reader)
	if !ok {
		return nil, errors.New("zst must be used with a seekable input")
	}
	seeker, err := storage.NewSeeker(reader)
	if err != nil {
		return nil, errors.New("zst must be used with a seekable input")
	}
	return zst.NewReaderFromSeeker(zctx, seeker)
}
