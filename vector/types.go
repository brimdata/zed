package vector

import (
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
)

type vector interface {
	// Returned materializer may panic if called more than `Vector.Length` times.
	// Use `Vector.NewMaterializer` for a safe interface.
	newMaterializer() materializer
}

var _ vector = (*bools)(nil)
var _ vector = (*byteses)(nil)
var _ vector = (*durations)(nil)
var _ vector = (*float16s)(nil)
var _ vector = (*float32s)(nil)
var _ vector = (*float64s)(nil)
var _ vector = (*int8s)(nil)
var _ vector = (*int16s)(nil)
var _ vector = (*int32s)(nil)
var _ vector = (*int64s)(nil)
var _ vector = (*ips)(nil)
var _ vector = (*nets)(nil)
var _ vector = (*strings)(nil)
var _ vector = (*times)(nil)
var _ vector = (*types)(nil)
var _ vector = (*uint8s)(nil)
var _ vector = (*uint16s)(nil)
var _ vector = (*uint32s)(nil)
var _ vector = (*uint64s)(nil)

var _ vector = (*arrays)(nil)
var _ vector = (*constants)(nil)
var _ vector = (*maps)(nil)
var _ vector = (*nulls)(nil)
var _ vector = (*records)(nil)
var _ vector = (*unions)(nil)

// TODO Use bitset.
type bools struct {
	values []bool
}

type byteses struct {
	data []byte
	// offsets[0] == 0
	// len(offsets) == len(vector) + 1
	offsets []int
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

type int8s struct {
	values []int8
}

type int16s struct {
	values []int16
}

type int32s struct {
	values []int32
}

type int64s struct {
	values []int64
}

type ips struct {
	values []netip.Addr
}

type nets struct {
	values []netip.Prefix
}

type strings struct {
	data []byte
	// offsets[0] == 0
	// len(offsets) == len(vector) + 1
	offsets []int
}

type times struct {
	values []nano.Ts
}

type types struct {
	values []zed.Type
}

type uint8s struct {
	values []uint8
}

type uint16s struct {
	values []uint16
}

type uint32s struct {
	values []uint32
}

type uint64s struct {
	values []uint64
}

type arrays struct {
	lengths []int64
	elems   vector
}

type constants struct {
	bytes []byte
}

type maps struct {
	lengths []int64
	keys    vector
	values  vector
}

type nulls struct {
	// len(runs) > 0
	runs   []int64
	values vector
}

type records struct {
	fields []vector
}

type unions struct {
	payloads []vector
	tags     []int64
}
