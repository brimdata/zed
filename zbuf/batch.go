package zbuf

import (
	"sync"

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
	Vars() []zed.Value
}

func WrapBatch(b Batch, arena *zed.Arena, vals []zed.Value) Batch {
	panic("xxx to do")
}

type batch struct {
	vals *zed.ArenaValues
	// Vars holds variables reachable in the current scope.
	vars *zed.ArenaValues
}

func NewBatch(arena *zed.Arena, vals []zed.Value, vars *zed.ArenaValues) Batch {
	if vars == nil {
		vars = &zed.ArenaValues{}
	}
	return &batch{&zed.ArenaValues{Arena: arena, Values: vals}, vars}
}

func (b *batch) Ref()                {}
func (b *batch) Unref()              {}
func (b *batch) Values() []zed.Value { return b.vals.Values }
func (b *batch) Vars() []zed.Value   { return b.vars.Values }

// WriteBatch writes the values in batch to zw.  If an error occurs, WriteBatch
// stops and returns the error.
func WriteBatch(zw zio.Writer, batch Batch) error {
	for _, val := range batch.Values() {
		if err := zw.Write(val); err != nil {
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
	return &puller{zctx: zctx, zr: zr}
}

type puller struct {
	zctx      *zed.Context
	zr        zio.Reader
	arenaPool sync.Pool
}

func (p *puller) newArena() *zed.Arena {
	arena := p.arenaPool.Get().(*zed.Arena)
	if arena == nil {
		return zed.NewArenaInPool(p.zctx, &p.arenaPool)
	}
	arena.Ref()
	arena.Reset()
	return arena
}

func (p *puller) Pull(bool) (Batch, error) {
	if p.zr == nil {
		return nil, nil
	}
	arena := p.newArena()
	vals := make([]zed.Value, 0, PullerBatchValues)
	for {
		val, err := p.zr.Read(arena)
		if err != nil {
			return nil, err
		}
		if val == nil {
			p.zr = nil
			if len(vals) == 0 {
				return nil, nil
			}
			return NewBatch(arena, vals, nil), nil
		}
		vals = append(vals, *val)
		if len(vals) >= PullerBatchValues {
			return NewBatch(arena, vals, nil), nil
		}
	}
}

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

func (r *pullerReader) Read(arena *zed.Arena) (*zed.Value, error) {
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
	return val.CopyToArena(arena).Ptr(), nil
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
