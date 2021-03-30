package emitter

import (
	"bytes"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/detector"
	"github.com/brimdata/zed/zng/resolver"
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
	w, err := detector.LookupWriter(zio.NopCloser(&b.buf), resolver.NewContext(), opts)
	if err != nil {
		return nil, err
	}
	b.Writer = w
	return b, nil
}
