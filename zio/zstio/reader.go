package zstio

import (
	"errors"
	"io"

	"github.com/brimdata/zq/zng/resolver"
	"github.com/brimdata/zq/zst"
)

func NewReader(r io.Reader, zctx *resolver.Context) (*zst.Reader, error) {
	seeker, ok := r.(zst.Seeker)
	if !ok {
		return nil, errors.New("zst must be used with a seekable input")
	}
	return zst.NewReaderFromSeeker(zctx, seeker)
}
