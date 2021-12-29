package zngio

import (
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type batch struct {
	buf  *buffer
	refs int32
	vals []zed.Value
}

var _ zbuf.Batch = (*batch)(nil)

var batchPool sync.Pool

func newBatch(buf *buffer) *batch {
	b, ok := batchPool.Get().(*batch)
	if !ok {
		b = &batch{vals: make([]zed.Value, 200)}
	}
	b.buf = buf
	b.refs = 1
	b.vals = b.vals[:0]
	return b
}

func (b *batch) add(r *zed.Value) { b.vals = append(b.vals, *r) }

func (b *batch) Values() []zed.Value { return b.vals }

//XXX
func (b *batch) NewValue(typ zed.Type, bytes zcode.Bytes) *zed.Value {
	return zed.NewValue(typ, bytes)
}

func (b *batch) CopyValue(val zed.Value) *zed.Value {
	return zed.NewValue(val.Type, val.Bytes)
}

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

//XXX this should be ok, but we should handle nil receiver in scope so push
// will do the right thing
func (*batch) Vars() []zed.Value { return nil }

func (b *batch) Context() expr.Context { return b }
