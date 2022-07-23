package column

import (
	"io"

	"github.com/brimdata/zed"
)

type IntWriter struct {
	PrimitiveWriter
}

func NewIntWriter(spiller *Spiller) *IntWriter {
	return &IntWriter{*NewPrimitiveWriter(zed.TypeInt32, spiller)}
}

func (p *IntWriter) Write(v int32) error {
	return p.PrimitiveWriter.Write(zed.EncodeInt(int64(v)))
}

type IntReader struct {
	PrimitiveReader
}

func NewIntReader(segmap []Segment, r io.ReaderAt) *IntReader {
	return &IntReader{*NewPrimitiveReader(&Primitive{zed.TypeInt64, segmap}, r)}
}

func (p *IntReader) Read() (int64, error) {
	zv, err := p.read()
	if err != nil {
		return 0, err
	}
	return zed.DecodeInt(zv), err
}
