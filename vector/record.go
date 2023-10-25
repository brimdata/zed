package vector

import (
	"github.com/brimdata/zed"
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
	return nil
}
