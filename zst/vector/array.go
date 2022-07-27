package vector

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type ArrayWriter struct {
	typ     zed.Type
	values  Writer
	lengths *Int64Writer
}

func NewArrayWriter(inner zed.Type, spiller *Spiller) *ArrayWriter {
	return &ArrayWriter{
		typ:     inner,
		values:  NewWriter(inner, spiller),
		lengths: NewInt64Writer(spiller),
	}
}

func (a *ArrayWriter) Write(body zcode.Bytes) error {
	it := body.Iter()
	var len int64
	for !it.Done() {
		if err := a.values.Write(it.Next()); err != nil {
			return err
		}
		len++
	}
	return a.lengths.Write(len)
}

func (a *ArrayWriter) Flush(eof bool) error {
	if err := a.lengths.Flush(eof); err != nil {
		return err
	}
	return a.values.Flush(eof)
}

func (a *ArrayWriter) Metadata() Metadata {
	return &Array{
		Lengths: a.lengths.Segmap(),
		Values:  a.values.Metadata(),
	}
}

type ArrayReader struct {
	elems   Reader
	lengths *Int64Reader
}

func NewArrayReader(array *Array, r io.ReaderAt) (*ArrayReader, error) {
	elems, err := NewReader(array.Values, r)
	if err != nil {
		return nil, err
	}
	return &ArrayReader{
		elems:   elems,
		lengths: NewInt64Reader(array.Lengths, r),
	}, nil
}

func (a *ArrayReader) Read(b *zcode.Builder) error {
	len, err := a.lengths.Read()
	if err != nil {
		return err
	}
	b.BeginContainer()
	for k := 0; k < int(len); k++ {
		if err := a.elems.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

type SetWriter struct {
	ArrayWriter
}

func NewSetWriter(inner zed.Type, spiller *Spiller) *SetWriter {
	return &SetWriter{
		ArrayWriter{
			typ:     inner,
			values:  NewWriter(inner, spiller),
			lengths: NewInt64Writer(spiller),
		},
	}
}

func (s *SetWriter) Metadata() Metadata {
	return &Set{
		Lengths: s.lengths.Segmap(),
		Values:  s.values.Metadata(),
	}
}
