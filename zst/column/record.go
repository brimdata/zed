package column

import (
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

var ErrColumnMismatch = errors.New("zng record value doesn't match column writer")

type RecordWriter []*FieldWriter

func NewRecordWriter(typ *zed.TypeRecord, spiller *Spiller) RecordWriter {
	var r RecordWriter
	for _, col := range typ.Columns {
		fw := &FieldWriter{
			name:     col.Name,
			column:   NewWriter(col.Type, spiller),
			presence: NewPresenceWriter(spiller),
		}
		r = append(r, fw)
	}
	return r
}

func (r RecordWriter) Write(body zcode.Bytes) error {
	it := body.Iter()
	for _, f := range r {
		if it.Done() {
			return ErrColumnMismatch
		}
		body, _ := it.Next()
		if err := f.write(body); err != nil {
			return err
		}
	}
	if !it.Done() {
		return ErrColumnMismatch
	}
	return nil
}

func (r RecordWriter) Flush(eof bool) error {
	// XXX we might want to arrange these flushes differently for locality
	for _, f := range r {
		if err := f.Flush(eof); err != nil {
			return err
		}
	}
	return nil
}

func (r RecordWriter) MarshalZNG(zctx *zed.Context, b *zcode.Builder) (zed.Type, error) {
	var columns []zed.Column
	b.BeginContainer()
	for _, f := range r {
		fieldType, err := f.MarshalZNG(zctx, b)
		if err != nil {
			return nil, err
		}
		columns = append(columns, zed.Column{f.name, fieldType})
	}
	b.EndContainer()
	return zctx.LookupTypeRecord(columns)
}

type Record []*Field

func (r *Record) UnmarshalZNG(typ *zed.TypeRecord, in zed.Value, reader io.ReaderAt) error {
	rtype, ok := in.Type.(*zed.TypeRecord)
	if !ok {
		return errors.New("corrupt zst object: record_column is not a record")
	}
	k := 0
	for it := in.Bytes.Iter(); !it.Done(); k++ {
		if k >= len(typ.Columns) {
			return errors.New("mismatch between record type and record_column") //XXX
		}
		fieldType := typ.Columns[k].Type
		zv, _ := it.Next()
		f := &Field{}
		if err := f.UnmarshalZNG(fieldType, zed.Value{rtype.Columns[k].Type, zv}, reader); err != nil {
			return err
		}
		*r = append(*r, f)
	}
	return nil
}

func (r Record) Read(b *zcode.Builder) error {
	b.BeginContainer()
	for _, f := range r {
		if err := f.Read(b); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

var ErrNonRecordAccess = errors.New("attempting to access a field in a non-record value")

func (r Record) Lookup(typ *zed.TypeRecord, fields []string) (zed.Type, Interface, error) {
	if len(fields) == 0 {
		panic("column.Record.Lookup cannot be called with an empty fields argument")
	}
	k, ok := typ.ColumnOfField(fields[0])
	if !ok {
		return nil, nil, zed.ErrMissing
	}
	t := typ.Columns[k].Type
	if len(fields) == 1 {
		return t, r[k], nil
	}
	typ, ok = t.(*zed.TypeRecord)
	if !ok {
		// This condition can happen when you are cutting id.foo and there
		// is a field "id" that isn't a record so cut should ignore it.
		return nil, nil, ErrNonRecordAccess
	}
	return r[k].column.(*Record).Lookup(typ, fields[1:])
}
