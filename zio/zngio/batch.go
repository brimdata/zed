package zngio

import (
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
)

type batch struct {
	buf  *buffer
	recs []zed.Record
	refs int32
}

var batchPool sync.Pool

func newBatch(buf *buffer) *batch {
	b, ok := batchPool.Get().(*batch)
	if !ok {
		b = &batch{recs: make([]zed.Record, 200)}
	}
	b.buf = buf
	b.recs = b.recs[:0]
	b.refs = 1
	return b
}

func (b *batch) add(r *zed.Record) { b.recs = append(b.recs, *r) }

func (b *batch) Index(i int) *zed.Record { return &b.recs[i] }

func (b *batch) Length() int { return len(b.recs) }

func (b *batch) Records() []*zed.Record {
	var recs []*zed.Record
	for i := range b.recs {
		recs = append(recs, &b.recs[i])
	}
	return recs
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
