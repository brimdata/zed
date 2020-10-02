package expr

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type result struct {
	buf zcode.Bytes
}

func (r *result) Int(v int64) zcode.Bytes {
	r.buf = zng.AppendInt(r.buf[:0], v)
	return r.buf
}

func (r *result) Uint(v uint64) zcode.Bytes {
	r.buf = zng.AppendUint(r.buf[:0], v)
	return r.buf
}

func (r *result) Float64(v float64) zcode.Bytes {
	r.buf = zng.AppendFloat64(r.buf[:0], v)
	return r.buf
}

func (r *result) Time(v nano.Ts) zcode.Bytes {
	r.buf = zng.AppendTime(r.buf[:0], v)
	return r.buf
}
