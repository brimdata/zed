package zngio

import (
	"slices"
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
)

type batch struct {
	arena *zed.Arena
	refs  int32
	vals  []zed.Value
}

var _ zbuf.Batch = (*batch)(nil)

var batchPool sync.Pool

func newBatch(zctx *zed.Context, nbytes, nvals int) *batch {
	b, ok := batchPool.Get().(*batch)
	if ok {
		if b.refs != 0 {
			panic("zngio: nonzero batch referece count")
		}
		b.arena.Reset()
		b.vals = b.vals[:0]
	} else {
		b = &batch{arena: zed.NewArena(zctx)}
	}
	b.arena.Grow(nbytes)
	b.vals = slices.Grow(b.vals, nvals)
	return b
}

func (b *batch) append(val zed.Value) {
	b.vals = append(b.vals, val)
}

func (b *batch) Arena() *zed.Arena { return b.arena }

func (b *batch) Ref() { atomic.AddInt32(&b.refs, 1) }

func (b *batch) Unref() {
	if refs := atomic.AddInt32(&b.refs, -1); refs == 0 {
		batchPool.Put(b)
	} else if refs < 0 {
		panic("zngio: negative batch reference count")
	}
}

func (b *batch) Values() []zed.Value { return b.vals }

// XXX this should be ok, but we should handle nil receiver in scope so push
// will do the right thing
func (*batch) Vars() []zed.Value { return nil }
