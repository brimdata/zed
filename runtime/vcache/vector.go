package vcache

import (
	"fmt"
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zst/vector"
)

type iterator func(*zcode.Builder) error

// Vector is the primary interface to in-memory sequences of Zed values
// representing the ZST vector format.  As we implement additional optimizations
// and various forms of pushdown, we will enhance this interface with
// corresponding methods.
type Vector interface {
	NewIter(io.ReaderAt) (iterator, error)
}

// NewVector converts a ZST metadata reader to its equivalent vector cache
// metadata manager.
func NewVector(meta vector.Metadata, r io.ReaderAt) (Vector, error) {
	switch meta := meta.(type) {
	case *vector.Named:
		return NewVector(meta.Values, r)
	case *vector.Record:
		return NewRecord(meta.Fields, r)
	case *vector.Primitive:
		return NewPrimitive(meta)
	case *vector.Array:
		return NewArray(meta, r)
	case *vector.Set:
		a := *(*vector.Array)(meta)
		return NewArray(&a, r)
	case *vector.Map:
		return NewMap(meta, r)
	case *vector.Union:
		return NewUnion(meta, r)
	case *vector.Nulls:
		values, err := NewVector(meta.Values, r)
		if err != nil {
			return nil, err
		}
		return NewNulls(meta, values, r)
	default:
		return nil, fmt.Errorf("vector cache: type %T not supported", meta)
	}
}

func Under(v Vector) Vector {
	for {
		if nulls, ok := v.(*Nulls); ok {
			v = nulls.values
			continue
		}
		return v
	}
}
