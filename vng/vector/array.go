package vector

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type ArrayWriter struct {
	typ     zed.Type
	values  Writer
	lengths *Int64Writer
	count   uint32
}

var _ Writer = (*ArrayWriter)(nil)

func NewArrayWriter(typ *zed.TypeArray) *ArrayWriter {
	return &ArrayWriter{
		typ:     typ.Type,
		values:  NewWriter(typ.Type),
		lengths: NewInt64Writer(),
	}
}

func (a *ArrayWriter) Write(body zcode.Bytes) {
	a.count++
	it := body.Iter()
	var len int64
	for !it.Done() {
		a.values.Write(it.Next())
		len++
	}
	a.lengths.Write(len)
}

func (a *ArrayWriter) Encode(group *errgroup.Group) {
	a.lengths.Encode(group)
	a.values.Encode(group)
}

func (a *ArrayWriter) Emit(w io.Writer) error {
	if err := a.lengths.Emit(w); err != nil {
		return err
	}
	return a.values.Emit(w)
}

func (a *ArrayWriter) Metadata(off uint64) (uint64, Metadata) {
	var lens, vals Metadata
	off, lens = a.lengths.Metadata(off)
	off, vals = a.values.Metadata(off)
	return off, &Array{
		Length:  a.count,
		Lengths: lens.(*Primitive).Location, //XXX
		Values:  vals,
	}
}

type ArrayReader struct {
	Elems   Reader
	Lengths *Int64Reader
}

func NewArrayReader(array *Array, r io.ReaderAt) (*ArrayReader, error) {
	elems, err := NewReader(array.Values, r)
	if err != nil {
		return nil, err
	}
	return &ArrayReader{
		Elems:   elems,
		Lengths: NewInt64Reader(array.Lengths, r),
	}, nil
}

func (a *ArrayReader) Read(b *zcode.Builder) error {
	len, err := a.Lengths.Read()
	if err != nil {
		return err
	}
	b.BeginContainer()
	for k := 0; k < int(len); k++ {
		if err := a.Elems.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

type SetWriter struct {
	ArrayWriter
}

func NewSetWriter(typ *zed.TypeSet) *SetWriter {
	return &SetWriter{
		ArrayWriter{
			typ:     typ.Type,
			values:  NewWriter(typ.Type),
			lengths: NewInt64Writer(),
		},
	}
}

func (s *SetWriter) Metadata(off uint64) (uint64, Metadata) {
	off, meta := s.ArrayWriter.Metadata(off)
	array := meta.(*Array)
	return off, &Set{
		Length:  array.Length,
		Lengths: array.Lengths,
		Values:  array.Values,
	}
}
