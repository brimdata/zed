package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// XXX need to create memory model
type mem struct{}

func (*mem) Ref()   {}
func (*mem) Unref() {}

type Record struct {
	mem
	Typ    *zed.TypeRecord
	Fields []Any
}

var _ Any = (*Record)(nil)

func NewRecord(typ *zed.TypeRecord) *Record {
	return NewRecordWithFields(typ, make([]Any, len(typ.Fields)))
}

func NewRecordWithFields(typ *zed.TypeRecord, fields []Any) *Record {
	return &Record{Typ: typ, Fields: fields}
}

func (r *Record) Type() zed.Type {
	return r.Typ
}

func (r *Record) NewBuilder() Builder {
	//XXX
	fields := make([]Builder, 0, len(r.Fields))
	for _, v := range r.Fields {
		fields = append(fields, v.NewBuilder())
	}
	//XXX should change Builder API to not return bool because
	// you should never be called if you would return a nil...
	// the top level needs to know how much stuff there is, no?
	// That said, we should be robust to file errors like bad runlens
	// and return an error instead of panic.
	return func(b *zcode.Builder) bool {
		b.BeginContainer()
		for _, f := range fields {
			if !f(b) {
				return false
			}
		}
		b.EndContainer()
		return true
	}
}

func (r *Record) Key(b []byte, slot int) []byte {
	panic("TBD")
}

func (r *Record) Length() int {
	if len(r.Fields) == 0 {
		//XXX need to handle vector of {}
		panic("TBD")
	}
	return r.Fields[0].Length()
}

func (r *Record) Serialize(slot int) *zed.Value {
	panic("TBD")
}
