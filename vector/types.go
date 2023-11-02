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
var _ vector = (*ints)(nil)
var _ vector = (*ips)(nil)
var _ vector = (*nets)(nil)
var _ vector = (*strings)(nil)
var _ vector = (*times)(nil)
var _ vector = (*types)(nil)
var _ vector = (*uints)(nil)

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

type types struct {
	values []zed.Type
}

type uints struct {
	values []uint64
}

type arrays struct {
	lengths []int64
	elems   vector
}

type constants struct {
	value zed.Value
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
