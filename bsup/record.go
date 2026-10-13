package bsup

import (
	"io"

	"github.com/superdb/super/vector"
	"golang.org/x/sync/errgroup"
)

type RecordEncoder struct {
	fields []*FieldEncoder
	count  uint32
}

var _ Encoder = (*RecordEncoder)(nil)

func NewRecordEncoder(bctx *Context, vec *vector.Record) *RecordEncoder {
	fields := make([]*FieldEncoder, 0, len(vec.Fields))
	for i, f := range vec.Fields {
		fields = append(fields, &FieldEncoder{
			name:   vec.Typ.Fields[i].Name,
			values: NewEncoder(bctx, f),
		})
	}
	return &RecordEncoder{fields: fields, count: vec.Len()}
}

func (r *RecordEncoder) Encode(group *errgroup.Group) {
	for _, f := range r.fields {
		f.Encode(group, r.count)
	}
}

func (r *RecordEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	fields := make([]Field, 0, len(r.fields))
	for _, field := range r.fields {
		next, m := field.Metadata(bctx, off)
		fields = append(fields, m)
		off = next
	}
	return off, bctx.enter(&Record{Length: r.count, Fields: fields})
}

func (r *RecordEncoder) Emit(w io.Writer) error {
	for _, f := range r.fields {
		if err := f.Emit(w); err != nil {
			return err
		}
	}
	return nil
}
