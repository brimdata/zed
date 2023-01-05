package vector

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/exp/slices"
)

type PrimitiveWriter struct {
	typ      zed.Type
	bytes    zcode.Bytes
	spiller  *Spiller
	segments []Segment
}

func NewPrimitiveWriter(typ zed.Type, spiller *Spiller) *PrimitiveWriter {
	return &PrimitiveWriter{
		typ:     typ,
		spiller: spiller,
	}
}

func (p *PrimitiveWriter) Write(body zcode.Bytes) error {
	p.bytes = zcode.Append(p.bytes, body)
	var err error
	if len(p.bytes) >= p.spiller.Thresh {
		err = p.Flush(false)
	}
	return err
}

func (p *PrimitiveWriter) Flush(eof bool) error {
	var err error
	if len(p.bytes) > 0 {
		p.segments, err = p.spiller.Write(p.segments, p.bytes)
		p.bytes = p.bytes[:0]
	}
	return err
}

func (p *PrimitiveWriter) Segmap() []Segment {
	return p.segments
}

func (p *PrimitiveWriter) Metadata() Metadata {
	return &Primitive{
		Typ:    p.typ,
		Segmap: p.segments,
	}
}

type PrimitiveReader struct {
	segmap []Segment
	reader io.ReaderAt

	buf []byte
	it  zcode.Iter
}

func NewPrimitiveReader(primitive *Primitive, reader io.ReaderAt) *PrimitiveReader {
	return &PrimitiveReader{
		reader: reader,
		segmap: primitive.Segmap,
	}
}

func (p *PrimitiveReader) Read(b *zcode.Builder) error {
	zv, err := p.read()
	if err == nil {
		b.Append(zv)
	}
	return err
}

func (p *PrimitiveReader) read() (zcode.Bytes, error) {
	if p.it == nil || p.it.Done() {
		if len(p.segmap) == 0 {
			return nil, io.EOF
		}
		if err := p.next(); err != nil {
			return nil, err
		}
	}
	return p.it.Next(), nil
}

func (p *PrimitiveReader) next() error {
	segment := p.segmap[0]
	p.segmap = p.segmap[1:]
	if segment.Length > 2*MaxSegmentThresh {
		return errors.New("segment too big")
	}
	p.buf = slices.Grow(p.buf[:0], int(segment.MemLength))[:segment.MemLength]
	if err := segment.Read(p.reader, p.buf); err != nil {
		return err
	}
	p.it = zcode.Iter(p.buf)
	return nil
}
