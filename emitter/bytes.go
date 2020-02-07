package emitter

import (
	"bytes"

	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
)

type Bytes struct {
	*zio.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(format string, flags *zio.Flags) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = detector.LookupWriter(format, &noClose{&b.buf}, flags)
	if b.Writer == nil {
		return nil, unknownFormat(format)
	}
	return b, nil
}
