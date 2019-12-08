package emitter

import (
	"bytes"

	"github.com/mccanne/zq/pkg/zio"
	"github.com/mccanne/zq/pkg/zio/textio"
)

type Bytes struct {
	*zio.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(format string, tc *textio.Config) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = zio.LookupWriter(format, &noClose{&b.buf}, tc)
	if b.Writer == nil {
		return nil, unknownFormat(format)
	}
	return b, nil
}
