package zstio

import (
	"errors"
	"io"

	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zst"
)

type Reader struct {
	zst.Reader
}

func NewReader(r io.Reader, zctx *resolver.Context) (*Reader, error) {
	seeker, ok := r.(zst.Seeker)
	if !ok {
		return nil, errors.New("zst must be used with a seekable input")
	}
	reader, err := zst.NewReaderFromSeeker(zctx, seeker)
	if err != nil {
		return nil, err
	}
	return &Reader{*reader}, nil
}
