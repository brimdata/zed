package storage

import (
	"bytes"
)

type bytesReader struct {
	*bytes.Reader
}

var _ Reader = (*bytesReader)(nil)
var _ Sizer = (*bytesReader)(nil)

func NewBytesReader(b []byte) *bytesReader {
	return &bytesReader{bytes.NewReader(b)}
}

func (*bytesReader) Close() error {
	return nil
}

func (b *bytesReader) Size() (int64, error) {
	return b.Reader.Size(), nil
}
