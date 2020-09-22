package zstio

import (
	"errors"
	"io"

	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zst"
)

func NewReader(r io.Reader, zctx *resolver.Context) (*zst.Reader, error) {
	seeker, ok := r.(zst.Seeker)
	if !ok {
		return nil, errors.New("zst must be used with a seekable input")
	}
	return zst.NewReaderFromSeeker(zctx, seeker)
}
