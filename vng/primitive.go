package vng

import (
	"fmt"
	"io"
	"slices"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

const MaxDictSize = 256

type PrimitiveEncoder struct {
	typ      zed.Type
	bytes    zcode.Bytes
	bytesLen uint64
	format   uint8
	out      []byte
	dict     map[string]uint32
	cmp      expr.CompareFn
	min      *zed.Value
	max      *zed.Value
	count    uint32

	arena    *zed.Arena
	minArena *zed.Arena
	maxArena *zed.Arena
}

func NewPrimitiveEncoder(typ zed.Type, useDict bool) *PrimitiveEncoder {
	var dict map[string]uint32
	if useDict {
		// Don't bother using a dictionary (which takes 8-bit tags) to encode
		// other 8-bit values.
		if id := typ.ID(); id != zed.IDUint8 && id != zed.IDInt8 && id != zed.IDBool {
			dict = make(map[string]uint32)
		}
	}
	return &PrimitiveEncoder{
		typ:  typ,
		dict: dict,
		cmp:  expr.NewValueCompareFn(order.Asc, false),

		arena:    zed.NewArena(),
		minArena: zed.NewArena(),
		maxArena: zed.NewArena(),
	}
}

func (p *PrimitiveEncoder) Write(body zcode.Bytes) {
	p.update(body)
	p.bytes = zcode.Append(p.bytes, body)
}

func (p *PrimitiveEncoder) update(body zcode.Bytes) {
	p.count++
	if body == nil {
		panic("PrimitiveWriter should not be called with null")
	}
	p.arena.Reset()
	val := p.arena.New(p.typ, body)
	if p.min == nil || p.cmp(val, *p.min) < 0 {
		p.minArena.Reset()
		p.min = val.Copy(p.minArena).Ptr()
	}
	if p.max == nil || p.cmp(val, *p.max) > 0 {
		p.maxArena.Reset()
		p.max = val.Copy(p.maxArena).Ptr()
	}
	if p.dict != nil {
		p.dict[string(body)]++
		if len(p.dict) > MaxDictSize {
			p.dict = nil
		}
	}
}

func (p *PrimitiveEncoder) Encode(group *errgroup.Group) {
	group.Go(func() error {
		if p.dict != nil {
			p.bytes = p.makeDictVector()
		}
		fmt, out, err := compressBuffer(p.bytes)
		if err != nil {
			return err
		}
		p.format = fmt
		p.out = out
		p.bytesLen = uint64(len(p.bytes))
		p.bytes = nil // send to GC
		return nil
	})
}

func (p *PrimitiveEncoder) makeDictVector() []byte {
	dict, arena := p.makeDict()
	defer arena.Unref()
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

func (p *PrimitiveEncoder) Const() *Const {
	if len(p.dict) != 1 {
		return nil
	}
	var bytes zcode.Bytes
	if len(p.dict) == 1 {
		for b := range p.dict {
			bytes = []byte(b)
		}
	}
	arena := zed.NewArena()
	return &Const{
		Arena: arena,
		Value: arena.New(p.typ, bytes),
		Count: p.count,
	}
}

func (p *PrimitiveEncoder) Metadata(off uint64) (uint64, Metadata) {
	var dict []DictEntry
	var dictArena *zed.Arena
	if p.dict != nil {
		if cnt := len(p.dict); cnt != 0 {
			if cnt == 1 {
				// A Const vector takes no space in the data area so we
				// return off unmodified.  We also clear the output so
				// Emit does not write the one value in the vector.
				p.out = nil
				return off, p.Const()
			}
			dict, dictArena = p.makeDict()
		}
	}
	loc := Segment{
		Offset:            int64(off),
		Length:            int32(len(p.out)),
		MemLength:         int32(p.bytesLen),
		CompressionFormat: p.format,
	}
	off += uint64(len(p.out))
	return off, &Primitive{
		Typ:       p.typ,
		Location:  loc,
		Dict:      dict,
		Count:     p.count,
		Min:       p.min,
		Max:       p.max,
		dictArena: dictArena,
		minArena:  p.minArena,
		maxArena:  p.maxArena,
	}
}

func (p *PrimitiveEncoder) Emit(w io.Writer) error {
	var err error
	if len(p.out) > 0 {
		_, err = w.Write(p.out)
	}
	return err
}

func (p *PrimitiveEncoder) makeDict() ([]DictEntry, *zed.Arena) {
	arena := zed.NewArena()
	dict := make([]DictEntry, 0, len(p.dict))
	for key, cnt := range p.dict {
		dict = append(dict, DictEntry{
			arena.New(p.typ, zcode.Bytes(key)),
			cnt,
		})
	}
	sortDict(dict, p.cmp)
	return dict, arena
}

func sortDict(entries []DictEntry, cmp expr.CompareFn) {
	sort.Slice(entries, func(i, j int) bool {
		return cmp(entries[i].Value, entries[j].Value) < 0
	})
}

type PrimitiveBuilder struct {
	Typ zed.Type

	loc    Segment
	reader io.ReaderAt

	buf []byte
	it  zcode.Iter
}

func NewPrimitiveBuilder(primitive *Primitive, reader io.ReaderAt) *PrimitiveBuilder {
	return &PrimitiveBuilder{
		Typ:    primitive.Typ,
		reader: reader,
		loc:    primitive.Location,
	}
}

func (p *PrimitiveBuilder) Build(b *zcode.Builder) error {
	zv, err := p.ReadBytes()
	if err == nil {
		b.Append(zv)
	}
	return err
}

func (p *PrimitiveBuilder) ReadBytes() (zcode.Bytes, error) {
	if p.buf == nil {
		p.buf = make([]byte, p.loc.MemLength)
		if err := p.loc.Read(p.reader, p.buf); err != nil {
			return nil, err
		}
		p.it = zcode.Iter(p.buf)
	}
	if p.it == nil || p.it.Done() {
		return nil, io.EOF
	}
	return p.it.Next(), nil
}

type DictBuilder struct {
	Typ zed.Type

	loc       Segment
	reader    io.ReaderAt
	dict      []DictEntry
	selectors []byte
	off       int

	dictArena *zed.Arena
}

var _ Builder = (*DictBuilder)(nil)

func NewDictBuilder(primitive *Primitive, reader io.ReaderAt) *DictBuilder {
	return &DictBuilder{
		Typ:    primitive.Typ,
		reader: reader,
		loc:    primitive.Location,
		dict:   primitive.Dict,

		dictArena: primitive.dictArena,
	}
}

func (d *DictBuilder) Build(b *zcode.Builder) error {
	bytes, err := d.ReadBytes()
	if err == nil {
		b.Append(bytes)
	}
	return err
}

func (d *DictBuilder) ReadBytes() (zcode.Bytes, error) {
	if d.selectors == nil {
		d.selectors = make([]byte, d.loc.MemLength)
		if err := d.loc.Read(d.reader, d.selectors); err != nil {
			return nil, err
		}
	}
	if d.off >= len(d.selectors) {
		return nil, io.EOF
	}
	sel := int(d.selectors[d.off])
	d.off++
	if sel >= len(d.dict) {
		return nil, fmt.Errorf("corrupt VNG: selector (%d) out of range (len %d)", sel, len(d.dict))
	}
	return d.dict[sel].Value.Bytes(), nil
}

type ConstBuilder struct {
	Typ   zed.Type
	bytes zcode.Bytes
	cnt   uint32
}

var _ Builder = (*ConstBuilder)(nil)

func NewConstBuilder(c *Const) *ConstBuilder {
	return &ConstBuilder{Typ: c.Value.Type(), bytes: slices.Clone(c.Value.Bytes()), cnt: c.Count}
}

func (c *ConstBuilder) Build(b *zcode.Builder) error {
	if c.cnt == 0 {
		return io.EOF
	}
	c.cnt--
	b.Append(c.bytes)
	return nil
}
