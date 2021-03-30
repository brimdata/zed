package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
)

type ArrayWriter struct {
	typ     zng.Type
	values  Writer
	lengths *IntWriter
}

func NewArrayWriter(inner zng.Type, spiller *Spiller) *ArrayWriter {
	return &ArrayWriter{
		typ:     inner,
		values:  NewWriter(inner, spiller),
		lengths: NewIntWriter(spiller),
	}
}

func (a *ArrayWriter) Write(body zcode.Bytes) error {
	it := body.Iter()
	var len int32
	for !it.Done() {
		body, _, err := it.Next()
		if err != nil {
			return err
		}
		if err := a.values.Write(body); err != nil {
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

func (a *ArrayWriter) MarshalZNG(zctx *resolver.Context, b *zcode.Builder) (zng.Type, error) {
	b.BeginContainer()
	valType, err := a.values.MarshalZNG(zctx, b)
	if err != nil {
		return nil, err
	}
	lenType, err := a.lengths.MarshalZNG(zctx, b)
	if err != nil {
		return nil, err
	}
	b.EndContainer()
	cols := []zng.Column{
		{"values", valType},
		{"lengths", lenType},
	}
	return zctx.LookupTypeRecord(cols)
}

type Array struct {
	values  Interface
	lengths *Int
}

func (a *Array) UnmarshalZNG(inner zng.Type, in zng.Value, r io.ReaderAt) error {
	typ, ok := in.Type.(*zng.TypeRecord)
	if !ok {
		return errors.New("zst object array_column not a record")
	}
	rec := zng.NewRecord(typ, in.Bytes)
	zv, err := rec.Access("values")
	if err != nil {
		return err
	}
	a.values, err = Unmarshal(inner, zv, r)
	if err != nil {
		return err
	}
	zv, err = rec.Access("lengths")
	if err != nil {
		return err
	}
	a.lengths = &Int{}
	return a.lengths.UnmarshalZNG(zv, r)
}

func (a *Array) Read(b *zcode.Builder) error {
	len, err := a.lengths.Read()
	if err != nil {
		return err
	}
	b.BeginContainer()
	for k := 0; k < int(len); k++ {
		if err := a.values.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}
