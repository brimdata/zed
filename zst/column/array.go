package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type ArrayWriter struct {
	typ     zed.Type
	values  Writer
	lengths *IntWriter
}

func NewArrayWriter(inner zed.Type, spiller *Spiller) *ArrayWriter {
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

func (a *ArrayWriter) MarshalZNG(zctx *zed.Context, b *zcode.Builder) (zed.Type, error) {
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
	cols := []zed.Column{
		{"values", valType},
		{"lengths", lenType},
	}
	return zctx.LookupTypeRecord(cols)
}

type Array struct {
	values  Interface
	lengths *Int
}

func (a *Array) UnmarshalZNG(inner zed.Type, in zed.Value, r io.ReaderAt) error {
	typ, ok := in.Type.(*zed.TypeRecord)
	if !ok {
		return errors.New("zst object array_column not a record")
	}
	rec := zed.NewValue(typ, in.Bytes)
	zv, err := rec.Dot("values")
	if err != nil {
		return err
	}
	a.values, err = Unmarshal(inner, zv, r)
	if err != nil {
		return err
	}
	zv, err = rec.Dot("lengths")
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
