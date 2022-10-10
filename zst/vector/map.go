package vector

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type MapWriter struct {
	typ     *zed.TypeMap
	keys    Writer
	values  Writer
	lengths *Int64Writer
}

func NewMapWriter(typ *zed.TypeMap, spiller *Spiller) *MapWriter {
	return &MapWriter{
		typ:     typ,
		keys:    NewWriter(typ.KeyType, spiller),
		values:  NewWriter(typ.ValType, spiller),
		lengths: NewInt64Writer(spiller),
	}
}

func (m *MapWriter) Write(body zcode.Bytes) error {
	var len int
	it := body.Iter()
	for !it.Done() {
		keyBytes := it.Next()
		if it.Done() {
			return errors.New("zst writer: truncated map value")
		}
		valBytes := it.Next()
		if err := m.keys.Write(keyBytes); err != nil {
			return err
		}
		if err := m.values.Write(valBytes); err != nil {
			return err
		}
		len++
	}
	return m.lengths.Write(int64(len))
}

func (m *MapWriter) Flush(eof bool) error {
	if err := m.lengths.Flush(eof); err != nil {
		return err
	}
	if err := m.keys.Flush(eof); err != nil {
		return err
	}
	return m.values.Flush(eof)
}

func (m *MapWriter) Metadata() Metadata {
	return &Map{
		Lengths: m.lengths.Segmap(),
		Keys:    m.keys.Metadata(),
		Values:  m.values.Metadata(),
	}
}

type MapReader struct {
	keys    Reader
	values  Reader
	lengths *Int64Reader
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
		keys:    keys,
		values:  values,
		lengths: NewInt64Reader(m.Lengths, r),
	}, nil
}

func (m *MapReader) Read(b *zcode.Builder) error {
	len, err := m.lengths.Read()
	if err != nil {
		return err
	}
	b.BeginContainer()
	for k := 0; k < int(len); k++ {
		if err := m.keys.Read(b); err != nil {
			return err
		}
		if err := m.values.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}
