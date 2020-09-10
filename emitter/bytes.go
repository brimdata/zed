package emitter

import (
	"bytes"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
)

type Bytes struct {
	zbuf.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(flags *zio.WriterFlags) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = detector.LookupWriter(zio.NopCloser(&b.buf), flags)
	if b.Writer == nil {
		return nil, unknownFormat(flags.Format)
	}
	return b, nil
}
