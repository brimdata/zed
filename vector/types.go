package vector

import (
	"github.com/brimdata/zed"
)

// TODO How to handle (u)ints larger than 64 bits?

type any interface {
	// Returned materializer may panic if called more than `Vector.Length` times.
	// Use `Vector.NewMaterializer` for a safe interface.
	newMaterializer() materializer
}

var _ any = (*bools)(nil)
var _ any = (*ints)(nil)
var _ any = (*strings)(nil)
var _ any = (*uints)(nil)

var _ any = (*arrays)(nil)
var _ any = (*constants)(nil)
var _ any = (*maps)(nil)
var _ any = (*nulls)(nil)
var _ any = (*records)(nil)
var _ any = (*unions)(nil)

type bools struct {
	values []bool // TODO Use bitset.
}

type ints struct {
	values []int64
}

type strings struct {
	values []string
}

type uints struct {
	values []uint64
}

type arrays struct {
	lengths []int64
	elems   any
}

type constants struct {
	value zed.Value
}

type maps struct {
	lengths []int64
	keys    any
	values  any
}

type nulls struct {
	mask   []byte
	values any
}

type records struct {
	fields []any
}

type unions struct {
	payloads []any
	tags     []int64
}

func (vector *nulls) Has(index int64) bool {
	maskIndex := index >> 3
	bitIndex := index & 0b111
	return (vector.mask[maskIndex] & (1 << bitIndex)) != 0
}
