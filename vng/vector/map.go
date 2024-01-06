package vector

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type MapWriter struct {
	keys    Writer
	values  Writer
	lengths *Int64Writer
	count   uint32
}

func NewMapWriter(typ *zed.TypeMap) *MapWriter {
	return &MapWriter{
		keys:    NewWriter(typ.KeyType),
		values:  NewWriter(typ.ValType),
		lengths: NewInt64Writer(),
	}
}

func (m *MapWriter) Write(body zcode.Bytes) {
	m.count++
	var len int
	it := body.Iter()
	for !it.Done() {
		m.keys.Write(it.Next())
		m.values.Write(it.Next())
		len++
	}
	m.lengths.Write(int64(len))
}

func (m *MapWriter) Emit(w io.Writer) error {
	if err := m.lengths.Emit(w); err != nil {
		return err
	}
	if err := m.keys.Emit(w); err != nil {
		return err
	}
	return m.values.Emit(w)
}

func (m *MapWriter) Metadata(off uint64) (uint64, Metadata) {
	off, lens := m.lengths.Metadata(off)
	var keys, vals Metadata
	off, keys = m.keys.Metadata(off)
	off, vals = m.values.Metadata(off)
	return off, &Map{
		Lengths: lens.(*Primitive).Location,
		Keys:    keys,
		Values:  vals,
		Length:  m.count,
	}
}

func (m *MapWriter) Encode(group *errgroup.Group) {
	m.lengths.Encode(group)
	m.keys.Encode(group)
	m.values.Encode(group)
}

type MapReader struct {
	Keys    Reader
	Values  Reader
	Lengths *Int64Reader
}

func NewMapReader(m *Map, r io.ReaderAt) (*MapReader, error) {
	keys, err := NewReader(m.Keys, r)
	if err != nil {
		return nil, err
	}
	values, err := NewReader(m.Values, r)
	if err != nil {
		return nil, err
	}
	return &MapReader{
		Keys:    keys,
		Values:  values,
		Lengths: NewInt64Reader(m.Lengths, r),
	}, nil
}

func (m *MapReader) Read(b *zcode.Builder) error {
	len, err := m.Lengths.Read()
	if err != nil {
		return err
	}
	b.BeginContainer()
	for k := 0; k < int(len); k++ {
		if err := m.Keys.Read(b); err != nil {
			return err
		}
		if err := m.Values.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}
