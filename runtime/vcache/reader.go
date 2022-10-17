package vcache

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

type Reader struct {
	object  *Object
	iters   []iterator
	off     int
	builder zcode.Builder
}

var _ zio.Reader = (*Reader)(nil)

func (r *Reader) Read() (*zed.Value, error) {
	o := r.object
	if r.off >= len(o.typeIDs) {
		return nil, nil
	}
	id := o.typeIDs[r.off]
	r.off++
	it := r.iters[id]
	if it == nil {
		var err error
		it, err = o.vectors[id].NewIter(o.reader)
		if err != nil {
			return nil, err
		}
		r.iters[id] = it
	}
	r.builder.Reset()
	if err := it(&r.builder); err != nil {
		return nil, err
	}
	return zed.NewValue(o.types[id], r.builder.Bytes().Body()), nil
}
