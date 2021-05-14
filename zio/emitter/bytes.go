package emitter

import (
	"bytes"

	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
)

type Bytes struct {
	zio.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(opts anyio.WriterOpts) (*Bytes, error) {
	b := &Bytes{}
	w, err := anyio.NewWriter(zio.NopCloser(&b.buf), opts)
	if err != nil {
		return nil, err
	}
	b.Writer = w
	return b, nil
}
