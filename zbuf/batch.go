package zbuf

import (
	"slices"
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
)

// Batch is an interface to a bundle of values.  Reference counting allows
// efficient, safe reuse in concert with sharing across goroutines.
//
// A newly obtained Batch always has a reference count of one.  The Batch owns
// its values and their storage, and an implementation may reuse this memory
// when the reference count falls to zero, reducing load on the garbage
// collector.
//
// To promote reuse, a goroutine should release a Batch reference when possible,
// but care must be taken to avoid race conditions that arise from releasing a
// reference too soon.  Specifically, a goroutine obtaining a value from a
// Batch must retain its Batch reference for as long as it retains the value,
// and the goroutine must not use the value after releasing its reference.
//
// Regardless of reference count or implementation, an unreachable Batch will
// eventually be reclaimed by the garbage collector.
type Batch interface {
	Ref()
	Unref()
	Values() []zed.Value
	// Vars accesses the variables reachable in the current scope.
	Vars() []zed.Value
}

func NewBatch(parent Batch, arena *zed.Arena, values []zed.Value) Batch {
	return &batchWithArena{1, parent, arena, values, parent.Vars()}
}

type batchWithArena struct {
	refs   int32
	parent Batch
	arena  *zed.Arena
	values []zed.Value
	vars   []zed.Value
}

func (b *batchWithArena) Ref() { atomic.AddInt32(&b.refs, 1) }

func (b *batchWithArena) Unref() {
	if refs := atomic.AddInt32(&b.refs, 1); refs == 0 {
		b.parent.Unref()
		b.arena.Unref()
	} else if refs < 0 {
		panic(refs)
	}
}

func (b *batchWithArena) Values() []zed.Value { return b.values }
func (b *batchWithArena) Vars() []zed.Value   { return b.vars }

// WriteBatch writes the values in batch to zw.  If an error occurs, WriteBatch
// stops and returns the error.
func WriteBatch(zw zio.Writer, batch Batch) error {
	vals := batch.Values()
	for i := range vals {
		if err := zw.Write(vals[i]); err != nil {
			return err
		}
	}
	return nil
}

// A Puller produces Batches of records, signaling end-of-stream (EOS) by returning
// a nil Batch and nil error.  The done argument to Pull indicates that the stream
// should be terminated before its natural EOS.  An implementation must return
// EOS in response to a Pull call when the done parameter is true.  After seeing EOS,
// (either via done or its natural end), an implementation of an operator that
// processes pulled data should respond to additional calls to Pull as if restarting
// in its initial state.  For original sources of data (e.g., the from operator),
// once EOS is reached, additional calls to Pull after the first EOS should always
// return EOS.  Pull is not safe to call concurrently.
type Puller interface {
	Pull(bool) (Batch, error)
}

// PullerBatchBytes is the maximum number of bytes (in the zed.Value.Byte
// sense) per batch for a [Puller] created by [NewPuller].
const PullerBatchBytes = 512 * 1024 // xxx

// PullerBatchValues is the maximum number of values per batch for a [Puller]
// created by [NewPuller].
var PullerBatchValues = 100

// NewPuller returns a puller for zr that returns batches containing up to
// [PullerBatchBytes] bytes and [PullerBatchValues] values.
func NewPuller(zctx *zed.Context, zr zio.Reader) Puller {
	return &puller{zctx, zr}
}

type puller struct {
	zctx *zed.Context
	zr   zio.Reader
}

func (p *puller) Pull(bool) (Batch, error) {
	if p.zr == nil {
		return nil, nil
	}
	batch := newPullerBatch(p.zctx)
	for {
		val, err := p.zr.Read(batch.arena)
		if err != nil {
			return nil, err
		}
		if val == nil {
			p.zr = nil
			if len(batch.vals) == 0 {
				return nil, nil
			}
			return batch, nil
		}
		if batch.appendVal(*val) {
			return batch, nil
		}
	}
}

type pullerBatch struct {
	refs  atomic.Int32
	arena *zed.Arena
	vals  []zed.Value
}

var pullerBatchPool sync.Pool

func newPullerBatch(zctx *zed.Context) *pullerBatch {
	b, ok := pullerBatchPool.Get().(*pullerBatch)
	if ok {
		if b.arena.Zctx() != zctx {
			panic("zed.Context mismatch")
		}
		b.arena.Reset()
		b.vals = b.vals[:0]
	} else {
		b = &pullerBatch{arena: zed.NewArena(zctx)}
	}
	b.refs.Store(1)
	b.vals = slices.Grow(b.vals, PullerBatchValues)
	return b
}

// appendVal appends a copy of val to b.  appendVal returns true if b is full
// (i.e., b.buf is full, b.buf had insufficient space for val.Bytes, or b.val is
// full).  appendVal never reallocates b.buf or b.vals.
func (b *pullerBatch) appendVal(val zed.Value) bool {
	b.vals = append(b.vals, b.arena.NewValue(val.Type(), val.Bytes()))
	return len(b.vals) == cap(b.vals)
}

func (b *pullerBatch) Ref() { b.refs.Add(1) }

func (b *pullerBatch) Unref() {
	if refs := b.refs.Add(-1); refs == 0 {
		pullerBatchPool.Put(b)
	} else if refs < 0 {
		panic("zbuf: negative batch reference count")
	}
}

func (p *pullerBatch) Values() []zed.Value { return p.vals }
func (*pullerBatch) Vars() []zed.Value     { return nil }
func (p *pullerBatch) Zctx() *zed.Context  { return p.arena.Zctx() }

func CopyPuller(w zio.Writer, p Puller) error {
	for {
		b, err := p.Pull(false)
		if b == nil || err != nil {
			return err
		}
		if err := WriteBatch(w, b); err != nil {
			return err
		}
		b.Unref()
	}
}

func PullerReader(p Puller) zio.Reader {
	return &pullerReader{p: p}
}

type pullerReader struct {
	p     Puller
	batch Batch
	vals  []zed.Value
}

func (r *pullerReader) Read(*zed.Arena) (*zed.Value, error) {
	// Loop handles zero-length batches.
	for len(r.vals) == 0 {
		if r.batch != nil {
			r.batch.Unref()
			r.batch = nil
		}
		batch, err := r.p.Pull(false)
		if batch == nil || err != nil {
			return nil, err
		}
		r.batch = batch
		r.vals = batch.Values()
	}
	val := &r.vals[0]
	r.vals = r.vals[1:]
	return val, nil
}

func CopyVars(b Batch) []zed.Value {
	vars := b.Vars()
	if len(vars) > 0 {
		newvars := make([]zed.Value, len(vars))
		for k, v := range vars {
			newvars[k] = v.Copy()
		}
		vars = newvars
	}
	return vars
}

/*

type Batch2 struct {
	refs   atomic.Int32
	arena  *zed.Arena
	parent Batch
	vals   []zed.Value
}

var _ Batch = (*Batch2)(nil)
var _ expr.Context = (*Batch2)(nil)

var batch2Pool sync.Pool

func NewBatch2(parent Batch) *Batch2 {
	b, ok := batch2Pool.Get().(*Batch2)
	if ok {
		if b.arena.Zctx() != parent.Zctx() {
			panic("zed.Context mismatch")
		}
		b.vals = b.vals[:0]
	} else {
		b = &Batch2{arena: zed.NewArena(parent.Zctx())}
	}
	b.refs.Store(1)
	b.parent = parent
	b.vals = slices.Grow(b.vals, len(b.Values()))
	return b
}

func (b *Batch2) Append(val zed.Value) { b.vals = append(b.vals, val) }

func (b *Batch2) Arena() *zed.Arena { return b.arena }

func (b *Batch2) Ref() { b.refs.Add(1) }

func (b *Batch2) Unref() {
	if refs := b.refs.Add(-1); refs == 0 {
		b.parent.Unref()
		// Let parent be reclaimed separately.
		b.parent = nil
		pullerBatchPool.Put(b)
	} else if refs < 0 {
		panic("zbuf: negative batch reference count")
	}
}

func (b *Batch2) Values() []zed.Value { return b.vals }

func (b *Batch2) Vars() []zed.Value {
	if b.parent == nil {
		return nil
	}
	return b.parent.Vars()
}

func (b *Batch2) Zctx() *zed.Context { return b.arena.Zctx() }

*/
