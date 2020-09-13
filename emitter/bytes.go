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

func NewBytes(opts zio.WriterOpts) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = detector.LookupWriter(zio.NopCloser(&b.buf), opts)
	if b.Writer == nil {
		return nil, unknownFormat(opts.Format)
	}
	return b, nil
}
