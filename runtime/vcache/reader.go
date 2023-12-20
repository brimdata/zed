package vcache

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

type Reader struct {
	object   *Object
	builders []vector.Builder

	off     int
	builder zcode.Builder
	val     zed.Value
}

var _ zio.Reader = (*Reader)(nil)

func (r *Reader) Read() (*zed.Value, error) {
	o := r.object
	if r.off >= len(o.typeKeys) {
		return nil, nil
	}
	key := o.typeKeys[r.off]
	b := r.builders[key]
	if b == nil {
		vec, err := o.Load(uint32(key), nil)
		if err != nil {
			return nil, err
		}
		b = vec.NewBuilder()
		r.builders[key] = b
	}
	r.builder.Truncate()
	if !b(&r.builder) {
		panic(fmt.Sprintf("vector.Builder returned false for key %d at offset %d", key, r.off))
	}
	r.off++
	r.val = *zed.NewValue(o.typeDict[key], r.builder.Bytes().Body())
	return &r.val, nil
}
