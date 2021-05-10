package storage

import (
	"bytes"
)

type bytesReader struct {
	*bytes.Reader
}

var _ Reader = (*bytesReader)(nil)

func NewBytesReader(b []byte) *bytesReader {
	return &bytesReader{bytes.NewReader(b)}
}

func (*bytesReader) Close() error {
	return nil
}
