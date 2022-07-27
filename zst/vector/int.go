package vector

import (
	"io"

	"github.com/brimdata/zed"
)

type Int64Writer struct {
	PrimitiveWriter
}

func NewInt64Writer(spiller *Spiller) *Int64Writer {
	return &Int64Writer{*NewPrimitiveWriter(zed.TypeInt64, spiller)}
}

func (p *Int64Writer) Write(v int64) error {
	return p.PrimitiveWriter.Write(zed.EncodeInt(int64(v)))
}

type Int64Reader struct {
	PrimitiveReader
}

func NewInt64Reader(segmap []Segment, r io.ReaderAt) *Int64Reader {
	return &Int64Reader{*NewPrimitiveReader(&Primitive{zed.TypeInt64, segmap}, r)}
}

func (p *Int64Reader) Read() (int64, error) {
	zv, err := p.read()
	if err != nil {
		return 0, err
	}
	return zed.DecodeInt(zv), err
}
