package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Error struct {
	Typ   *zed.TypeError
	Vals  Any
	Nulls *Bool
}

var _ Any = (*Error)(nil)

// XXX we shouldn't create empty fields... this was the old design, now
// we create the entire vector structure and page in leaves, offsets, etc on demand
func NewError(typ *zed.TypeError, vals Any, nulls *Bool) *Error {
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

func NewStringError(zctx *zed.Context, msg string, len uint32) *Error {
	vals := NewConst(zed.NewString(msg), len, nil)
	return &Error{Typ: zctx.LookupTypeError(zed.TypeString), Vals: vals}
}

func NewMissing(zctx *zed.Context, len uint32) *Error {
	missing := zctx.Missing()
	vals := NewConst(missing, len, nil)
	return &Error{Typ: missing.Type().(*zed.TypeError), Vals: vals}
}

func NewWrappedError(zctx *zed.Context, msg string, val Any) *Error {
	recType := zctx.MustLookupTypeRecord([]zed.Field{
		{Name: "message", Type: zed.TypeString},
		{Name: "on", Type: val.Type()},
	})
	sval := NewConst(zed.NewString(msg), val.Len(), nil)
	rval := NewRecord(recType, []Any{sval, val}, val.Len(), nil)
	return &Error{Typ: zctx.LookupTypeError(recType), Vals: rval}
}
