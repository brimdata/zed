package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
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
	it     zcode.Iter
	segmap []Segment
	reader io.ReaderAt
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
	b := make([]byte, segment.Length)
	//XXX this where lots of seeks can happen until we put intelligent
	// scheduling in a layer below this informed by the reassembly maps
	// and the query that is going to run.
	n, err := p.reader.ReadAt(b, segment.Offset)
	if err != nil {
		return err
	}
	if n < int(segment.Length) {
		return errors.New("truncated read of ZST column")
	}
	p.it = zcode.Iter(b)
	return nil
}
