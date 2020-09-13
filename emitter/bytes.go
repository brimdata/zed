package emitter

import (
	"bytes"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/options"
)

type Bytes struct {
	zbuf.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(opts options.Writer) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = detector.LookupWriter(zio.NopCloser(&b.buf), opts)
	if b.Writer == nil {
		return nil, unknownFormat(opts.Format)
	}
	return b, nil
}
