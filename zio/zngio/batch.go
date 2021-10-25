package zngio

import (
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
)

type batch struct {
	buf   *buffer
	refs  int32
	zvals []zed.Value
}

var _ zbuf.Batch = (*batch)(nil)

var batchPool sync.Pool

func newBatch(buf *buffer) *batch {
	b, ok := batchPool.Get().(*batch)
	if !ok {
		b = &batch{zvals: make([]zed.Value, 200)}
	}
	b.buf = buf
	b.refs = 1
	b.zvals = b.zvals[:0]
	return b
}

func (b *batch) add(r *zed.Value) { b.zvals = append(b.zvals, *r) }

func (b *batch) Values() []zed.Value { return b.zvals }

func (b *batch) Ref() { atomic.AddInt32(&b.refs, 1) }

func (b *batch) Unref() {
	if refs := atomic.AddInt32(&b.refs, -1); refs == 0 {
		if b.buf != nil {
			b.buf.free()
			b.buf = nil
		}
		batchPool.Put(b)
	} else if refs < 0 {
		panic("zngio: negative batch reference count")
	}
}
