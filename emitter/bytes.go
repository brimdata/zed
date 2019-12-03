package emitter

import (
	"bytes"

	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zsio/text"
)

type Bytes struct {
	*zsio.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(format string, tc *text.Config) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = zsio.LookupWriter(format, &noClose{&b.buf}, tc)
	if b.Writer == nil {
		return nil, unknownFormat(format)
	}
	return b, nil
}
