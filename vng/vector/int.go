package vector

import (
	"io"

	"github.com/brimdata/zed"
)

type Int64Writer struct {
	PrimitiveWriter
}

func NewInt64Writer() *Int64Writer {
	return &Int64Writer{*NewPrimitiveWriter(zed.TypeInt64, false)}
}

func (p *Int64Writer) Write(v int64) {
	p.PrimitiveWriter.Write(zed.EncodeInt(v))
}

type Int64Reader struct {
	PrimitiveReader
}

func NewInt64Reader(loc Segment, r io.ReaderAt) *Int64Reader {
	return &Int64Reader{*NewPrimitiveReader(&Primitive{Typ: zed.TypeInt64, Location: loc}, r)}
}

func (p *Int64Reader) Read() (int64, error) {
	zv, err := p.ReadBytes()
	if err != nil {
		return 0, err
	}
	return zed.DecodeInt(zv), err
}
