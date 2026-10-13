package bsup

import (
	"github.com/superdb/super"
	"github.com/superdb/super/vector"
)

type EnumEncoder struct {
	*UintEncoder
	typ *super.TypeEnum
}

func NewEnumEncoder(vec *vector.Enum) *EnumEncoder {
	return &EnumEncoder{
		UintEncoder: &UintEncoder{typ: vec.Uint.Typ, vals: vec.Uint.Values},
		typ:         vec.Typ,
	}
}

func (e *EnumEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	off, values := e.UintEncoder.Metadata(bctx, off)
	return off, bctx.enter(&Enum{Symbols: e.typ.Symbols, Values: values})
}
