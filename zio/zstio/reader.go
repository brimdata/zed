package zstio

import (
	"errors"
	"io"

	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/zst"
)

func NewReader(r io.Reader, zctx *zson.Context) (*zst.Reader, error) {
	seeker, ok := r.(zst.Seeker)
	if !ok {
		return nil, errors.New("zst must be used with a seekable input")
	}
	return zst.NewReaderFromSeeker(zctx, seeker)
}
