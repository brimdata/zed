package emitter

import (
	"bytes"

	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zsio/detector"
)

type Bytes struct {
	*zsio.Writer
	buf bytes.Buffer
}

func (b *Bytes) Bytes() []byte {
	return b.buf.Bytes()
}

func NewBytes(format string, flags *zsio.Flags) (*Bytes, error) {
	b := &Bytes{}
	b.Writer = detector.LookupWriter(format, &noClose{&b.buf}, flags)
	if b.Writer == nil {
		return nil, unknownFormat(format)
	}
	return b, nil
}
