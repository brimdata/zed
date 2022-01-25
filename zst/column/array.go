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

func (a *ArrayWriter) EncodeMap(zctx *zed.Context, b *zcode.Builder) (zed.Type, error) {
	b.BeginContainer()
	valType, err := a.values.EncodeMap(zctx, b)
	if err != nil {
		return nil, err
	}
	lenType, err := a.lengths.EncodeMap(zctx, b)
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

type ArrayReader struct {
	elems   Reader
	lengths *IntReader
}

func NewArrayReader(inner zed.Type, in zed.Value, r io.ReaderAt) (*ArrayReader, error) {
	typ, ok := in.Type.(*zed.TypeRecord)
	if !ok {
		return nil, errors.New("ZST object array_column not a record")
	}
	rec := zed.NewValue(typ, in.Bytes)
	zv, err := rec.Access("values")
	if err != nil {
		return nil, err
	}
	elems, err := NewReader(inner, zv, r)
	if err != nil {
		return nil, err
	}
	zv, err = rec.Access("lengths")
	if err != nil {
		return nil, err
	}
	lengths, err := NewIntReader(zv, r)
	if err != nil {
		return nil, err
	}
	return &ArrayReader{
		elems:   elems,
		lengths: lengths,
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
