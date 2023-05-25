package vector

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/exp/slices"
)

const MaxDictSize = 256

type PrimitiveWriter struct {
	typ      zed.Type
	bytes    zcode.Bytes
	spiller  *Spiller
	segments []Segment
	dict     map[string]uint32
	cmp      expr.CompareFn
	min      *zed.Value
	max      *zed.Value
	count    uint32
	nulls    uint32
	hasNull  int
}

func NewPrimitiveWriter(typ zed.Type, spiller *Spiller, useDict bool) *PrimitiveWriter {
	var dict map[string]uint32
	if useDict {
		dict = make(map[string]uint32)
	}
	return &PrimitiveWriter{
		typ:     typ,
		spiller: spiller,
		dict:    dict,
		cmp:     expr.NewValueCompareFn(order.Asc, false),
	}
}

func (p *PrimitiveWriter) Write(body zcode.Bytes) error {
	p.update(body)
	p.bytes = zcode.Append(p.bytes, body)
	return nil
}

func (p *PrimitiveWriter) update(body zcode.Bytes) {
	p.count++
	if body == nil {
		p.nulls++
		p.hasNull = 1
		return
	}
	if body != nil {
		val := zed.NewValue(p.typ, body)
		if p.min == nil || p.cmp(val, p.min) < 0 {
			p.min = val
		}
		if p.max == nil || p.cmp(val, p.max) > 0 {
			p.max = val
		}
	}
	if p.dict != nil {
		p.dict[string(body)] += 1
		if len(p.dict)+p.hasNull > MaxDictSize {
			p.dict = nil
		}
	}
}

func (p *PrimitiveWriter) Flush(eof bool) error {
	if !eof {
		//XXX get rid of this... re-work flush protocol?
		panic("PrimitiveWriter.Flush")
	}
	if p.dict != nil {
		p.bytes = p.makeDictVector()
	}
	var err error
	if len(p.bytes) > 0 {
		p.segments, err = p.spiller.Write(p.segments, p.bytes)
		p.bytes = p.bytes[:0]
	}
	return err
}

func (p *PrimitiveWriter) makeDictVector() []byte {
	dict := p.makeDict()
	pos := make(map[string]byte)
	for off, entry := range dict {
		if bytes := entry.Value.Bytes(); bytes != nil {
			pos[string(bytes)] = byte(off)
		}
	}
	out := make([]byte, 0, p.count)
	for it := p.bytes.Iter(); !it.Done(); {
		bytes := it.Next()
		if bytes == nil {
			// null is always the first dict entry if it exists
			out = append(out, 0)
			continue
		}
		off, ok := pos[string(bytes)]
		if !ok {
			panic("bad dict entry") //XXX
		}
		out = append(out, off)
	}
	return out
}

func (p *PrimitiveWriter) Segmap() []Segment {
	return p.segments
}

func (p *PrimitiveWriter) Const() *Const {
	if len(p.dict)+p.hasNull != 1 {
		return nil
	}
	var bytes zcode.Bytes
	if len(p.dict) == 1 {
		for b := range p.dict {
			bytes = []byte(b)
		}
	}
	return &Const{
		Value: zed.NewValue(p.typ, bytes),
		Count: p.count,
	}
}

func (p *PrimitiveWriter) Metadata() Metadata {
	var dict []DictEntry
	if p.dict != nil {
		if cnt := len(p.dict) + p.hasNull; cnt != 0 {
			if cnt == 1 {
				return p.Const()
			}
			dict = p.makeDict()
		}
	}
	return &Primitive{
		Typ:    p.typ,
		Segmap: p.segments,
		Dict:   dict,
		Count:  p.count,
		Nulls:  p.nulls,
		Min:    p.min,
		Max:    p.max,
	}
}

func (p *PrimitiveWriter) makeDict() []DictEntry {
	dict := make([]DictEntry, 0, len(p.dict)+p.hasNull)
	for key, cnt := range p.dict {
		dict = append(dict, DictEntry{
			zed.NewValue(p.typ, zcode.Bytes(key)),
			uint32(cnt),
		})
	}
	if p.nulls != 0 {
		dict = append(dict, DictEntry{
			zed.NewValue(p.typ, nil),
			p.nulls,
		})
	}
	sortDict(dict, expr.NewValueCompareFn(order.Asc, false))
	return dict
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
		return errors.New("corrupt VNG: segment too big")
	}
	p.buf = slices.Grow(p.buf[:0], int(segment.MemLength))[:segment.MemLength]
	if err := segment.Read(p.reader, p.buf); err != nil {
		return err
	}
	p.it = zcode.Iter(p.buf)
	return nil
}

type DictReader struct {
	segmap    []Segment
	reader    io.ReaderAt
	dict      []DictEntry
	selectors []byte
	off       int
}

func NewDictReader(primitive *Primitive, reader io.ReaderAt) *DictReader {
	return &DictReader{
		reader: reader,
		segmap: primitive.Segmap,
		dict:   primitive.Dict,
	}
}

func (d *DictReader) Read(b *zcode.Builder) error {
	bytes, err := d.read()
	if err == nil {
		b.Append(bytes)
	}
	return err
}

func (d *DictReader) read() (zcode.Bytes, error) {
	if d.off >= len(d.selectors) {
		if len(d.segmap) == 0 {
			return nil, io.EOF
		}
		if err := d.next(); err != nil {
			return nil, err
		}
	}
	sel := int(d.selectors[d.off])
	d.off++
	if sel >= len(d.dict) {
		return nil, fmt.Errorf("corrupt VNG: selector (%d) out of range (len %d)", sel, len(d.dict))
	}
	return d.dict[sel].Value.Bytes(), nil
}

func (d *DictReader) next() error {
	segment := d.segmap[0]
	d.segmap = d.segmap[1:]
	if segment.Length > 2*MaxSegmentThresh {
		return errors.New("corrupt VNG: segment too big")
	}
	d.selectors = slices.Grow(d.selectors[:0], int(segment.MemLength))[:segment.MemLength]
	if err := segment.Read(d.reader, d.selectors); err != nil {
		return err
	}
	d.off = 0
	return nil
}

type ConstReader struct {
	bytes zcode.Bytes
	cnt   uint32
}

func NewConstReader(c *Const) *ConstReader {
	return &ConstReader{bytes: c.Value.Bytes(), cnt: c.Count}
}

func (c *ConstReader) Read(b *zcode.Builder) error {
	if c.cnt == 0 {
		return io.EOF
	}
	c.cnt--
	b.Append(c.bytes)
	return nil
}
