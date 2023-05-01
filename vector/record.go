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
	return &Record{Typ: typ, Fields: make([]Any, len(typ.Fields))}
}

func (r *Record) Type() zed.Type {
	return r.Typ
}
