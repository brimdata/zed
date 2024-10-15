package vng

import (
	"fmt"
	"io"

	"github.com/brimdata/super/zcode"
)

type Builder interface {
	Build(*zcode.Builder) error
}

func NewBuilder(meta Metadata, r io.ReaderAt) (Builder, error) {
	switch meta := meta.(type) {
	case nil:
		return nil, nil
	case *Nulls:
		inner, err := NewBuilder(meta.Values, r)
		if err != nil {
			return nil, err
		}
		return NewNullsBuilder(inner, meta.Runs, r), nil
	case *Named:
		return NewBuilder(meta.Values, r)
	case *Error:
		return NewBuilder(meta.Values, r)
	case *Record:
		return NewRecordBuilder(meta, r)
	case *Array:
		return NewArrayBuilder(meta, r)
	case *Set:
		return NewArrayBuilder((*Array)(meta), r)
	case *Map:
		return NewMapBuilder(meta, r)
	case *Union:
		return NewUnionBuilder(meta, r)
	case *Primitive:
		if len(meta.Dict) != 0 {
			return NewDictBuilder(meta, r), nil
		}
		return NewPrimitiveBuilder(meta, r), nil
	case *Const:
		return NewConstBuilder(meta), nil
	default:
		return nil, fmt.Errorf("unknown VNG metadata type: %T", meta)
	}
}
