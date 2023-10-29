package vector

import (
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"

	"github.com/RoaringBitmap/roaring"
)

type any interface {
	// Returned materializer may panic if called more than `Vector.Length` times.
	// Use `Vector.NewMaterializer` for a safe interface.
	newMaterializer() materializer
}

var _ any = (*bools)(nil)
var _ any = (*byteses)(nil)
var _ any = (*durations)(nil)
var _ any = (*float16s)(nil)
var _ any = (*float32s)(nil)
var _ any = (*float64s)(nil)
var _ any = (*ints)(nil)
var _ any = (*ips)(nil)
var _ any = (*nets)(nil)
var _ any = (*strings)(nil)
var _ any = (*times)(nil)
var _ any = (*uints)(nil)

var _ any = (*arrays)(nil)
var _ any = (*constants)(nil)
var _ any = (*maps)(nil)
var _ any = (*nulls)(nil)
var _ any = (*records)(nil)
var _ any = (*unions)(nil)

// TODO Use bitset.
type bools struct {
	values []bool
}

// TODO Read entire vector as single []byte.
type byteses struct {
	values [][]byte
}

type durations struct {
	values []nano.Duration
}

type float16s struct {
	// Not a typo - no native float16 type.
	// TODO Investigate https://pkg.go.dev/github.com/x448/float16
	values []float32
}

type float32s struct {
	values []float32
}

type float64s struct {
	values []float64
}

type ints struct {
	values []int64
}

type ips struct {
	values []netip.Addr
}

type nets struct {
	values []netip.Prefix
}

// TODO Read entire vector as single []byte.
type strings struct {
	values []string
}

type times struct {
	values []nano.Ts
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
	mask   *roaring.Bitmap
	values any
}

type records struct {
	fields []any
}

type unions struct {
	payloads []any
	tags     []int64
}
