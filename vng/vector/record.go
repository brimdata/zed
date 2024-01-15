package vector

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

var ErrVectorMismatch = errors.New("zng record value doesn't match vector writer")

type RecordWriter struct {
	fields []*FieldWriter
	count  uint32
}

var _ Writer = (*RecordWriter)(nil)

func NewRecordWriter(typ *zed.TypeRecord) *RecordWriter {
	fields := make([]*FieldWriter, 0, len(typ.Fields))
	for _, f := range typ.Fields {
		fields = append(fields, &FieldWriter{
			name:   f.Name,
			values: NewWriter(f.Type),
		})
	}
	return &RecordWriter{fields: fields}
}

func (r *RecordWriter) Write(body zcode.Bytes) {
	r.count++
	it := body.Iter()
	for _, f := range r.fields {
		f.write(it.Next())
	}
}

func (r *RecordWriter) Encode(group *errgroup.Group) {
	for _, f := range r.fields {
		f.Encode(group)
	}
}

func (r *RecordWriter) Metadata(off uint64) (uint64, Metadata) {
	fields := make([]Field, 0, len(r.fields))
	for _, field := range r.fields {
		next, m := field.Metadata(off)
		fields = append(fields, m)
		off = next
	}
	return off, &Record{Length: r.count, Fields: fields}
}

func (r *RecordWriter) Emit(w io.Writer) error {
	for _, f := range r.fields {
		if err := f.Emit(w); err != nil {
			return err
		}
	}
	return nil
}

type RecordReader struct {
	Names  []string
	Values []FieldReader
}

var _ Reader = (*RecordReader)(nil)

func NewRecordReader(record *Record, reader io.ReaderAt) (*RecordReader, error) {
	names := make([]string, 0, len(record.Fields))
	values := make([]FieldReader, 0, len(record.Fields))
	for _, field := range record.Fields {
		names = append(names, field.Name)
		fr, err := NewFieldReader(field, reader)
		if err != nil {
			return nil, err
		}
		values = append(values, *fr)
	}
	result := &RecordReader{
		Names:  names,
		Values: values,
	}
	return result, nil
}

func (r *RecordReader) Read(b *zcode.Builder) error {
	b.BeginContainer()
	for _, f := range r.Values {
		if err := f.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}
