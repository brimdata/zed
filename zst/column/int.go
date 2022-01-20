package column

import "github.com/brimdata/zed"

type IntWriter struct {
	PrimitiveWriter
}

func NewIntWriter(spiller *Spiller) *IntWriter {
	return &IntWriter{*NewPrimitiveWriter(spiller)}
}

func (p *IntWriter) Write(v int32) error {
	return p.PrimitiveWriter.Write(zed.EncodeInt(int64(v)))
}

type Int struct {
	Primitive
}

func (p *Int) Read() (int32, error) {
	zv, err := p.read()
	if err != nil {
		return 0, err
	}
	return int32(zed.DecodeInt(zv)), err
}
