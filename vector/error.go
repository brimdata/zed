package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Error struct {
	Typ   zed.Type
	Vals  Any
	Nulls *Bool
}

var _ Any = (*Error)(nil)

// XXX we shouldn't create empty fields... this was the old design, now
// we create the entire vector structure and page in leaves, offsets, etc on demand
func NewError(typ zed.Type, vals Any, nulls *Bool) *Error {
	return &Error{Typ: typ, Vals: vals, Nulls: nulls}
}

func (e *Error) Type() zed.Type {
	return e.Typ
}

func (e *Error) Len() uint32 {
	return e.Vals.Len()
}

func (e *Error) Serialize(b *zcode.Builder, slot uint32) {
	if e.Nulls != nil && e.Nulls.Value(slot) {
		b.Append(nil)
		return
	}
	e.Vals.Serialize(b, slot)

}

func NewMissing(zctx *zed.Context, len uint32) *Error {
	missing := zctx.Missing()
	vals := NewConst(missing, len, nil)
	return &Error{Typ: missing.Type(), Vals: vals}
}
