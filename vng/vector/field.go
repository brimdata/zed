package vector

import (
	"io"

	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type FieldWriter struct {
	name   string
	values Writer
}

func (f *FieldWriter) write(body zcode.Bytes) {
	f.values.Write(body)
}

func (f *FieldWriter) Metadata(off uint64) (uint64, Field) {
	off, meta := f.values.Metadata(off)
	return off, Field{
		Name:   f.name,
		Values: meta,
	}
}

func (f *FieldWriter) Encode(group *errgroup.Group) {
	f.values.Encode(group)
}

func (f *FieldWriter) Emit(w io.Writer) error {
	return f.values.Emit(w)
}

type FieldReader struct {
	Values Reader
}

func NewFieldReader(field Field, r io.ReaderAt) (*FieldReader, error) {
	values, err := NewReader(field.Values, r)
	if err != nil {
		return nil, err
	}
	return &FieldReader{
		Values: values,
	}, nil
}

func (f *FieldReader) Read(b *zcode.Builder) error {
	return f.Values.Read(b)
}
