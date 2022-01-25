package column

import (
	"io"

	"github.com/brimdata/zed"
)

type IntWriter struct {
	PrimitiveWriter
}

func NewIntWriter(spiller *Spiller) *IntWriter {
	return &IntWriter{*NewPrimitiveWriter(spiller)}
}

func (p *IntWriter) Write(v int32) error {
	return p.PrimitiveWriter.Write(zed.EncodeInt(int64(v)))
}

type IntReader struct {
	PrimitiveReader
}

func NewIntReader(val zed.Value, r io.ReaderAt) (*IntReader, error) {
	reader, err := NewPrimitiveReader(val, r)
	if err != nil {
		return nil, err
	}
	return &IntReader{*reader}, nil
}

func (p *IntReader) Read() (int64, error) {
	zv, err := p.read()
	if err != nil {
		return 0, err
	}
	return zed.DecodeInt(zv), err
}
