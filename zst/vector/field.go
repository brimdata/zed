package vector

import (
	"io"

	"github.com/brimdata/zed/zcode"
)

type FieldWriter struct {
	name   string
	values Writer
}

func (f *FieldWriter) write(body zcode.Bytes) error {
	return f.values.Write(body)
}

func (f *FieldWriter) Metadata() Field {
	return Field{
		Name:   f.name,
		Values: f.values.Metadata(),
	}
}

func (f *FieldWriter) Flush(eof bool) error {
	return f.values.Flush(eof)
}

type FieldReader struct {
	values Reader
}

func NewFieldReader(field Field, r io.ReaderAt) (*FieldReader, error) {
	values, err := NewReader(field.Values, r)
	if err != nil {
		return nil, err
	}
	return &FieldReader{
		values: values,
	}, nil
}

func (f *FieldReader) Read(b *zcode.Builder) error {
	return f.values.Read(b)
}
