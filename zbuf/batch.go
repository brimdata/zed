package zbuf

import (
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
	Vars() []zed.Value
}

type batch struct {
	refs    int32
	arena   *zed.Arena
	vals    []zed.Value
	batch   Batch
	vars    []zed.Value
	batches []Batch
	free    func()
}

func WrapBatch(b Batch, vals []zed.Value) Batch {
	return NewBatch(nil, vals, b, b.Vars())
}

func NewBatchWithVars(arena *zed.Arena, vals []zed.Value, vars []zed.Value) Batch {
	return NewBatch(arena, vals, nil, vars)
}

func NewBatchWithVarsAndFree(arena *zed.Arena, vals []zed.Value, vars []zed.Value, free func()) Batch {
	b := NewBatch(arena, vals, nil, vars)
	b.(*batch).free = free
	return b
}

func NewBatch(arena *zed.Arena, vals []zed.Value, b Batch, vars []zed.Value) Batch {
	if arena != nil {
		arena.Ref()
	}
	if b != nil {
		b.Ref()
	}
	return &batch{1, arena, vals, b, vars, nil, nil}
}

func (b *batch) AddBatches(batches ...Batch) {
	b.batches = append(b.batches, batches...)
}

func (b *batch) Ref() { atomic.AddInt32(&b.refs, 1) }

func (b *batch) Unref() {
	if refs := atomic.AddInt32(&b.refs, -1); refs == 0 {
		if b.arena != nil {
			b.arena.Unref()
		}
		if b.batch != nil {
			b.batch.Unref()
		}
		if b.free != nil {
			b.free()
		}
	} else if refs < 0 {
		panic("zbuf: negative batch reference count")
	}
}

func (b *batch) Values() []zed.Value { return b.vals }
func (b *batch) Vars() []zed.Value   { return b.vars }

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

// PullerBatchValues is the maximum number of values per batch for a [Puller]
// created by [NewPuller].
var PullerBatchValues = 100

// NewPuller returns a puller for zr that returns batches containing up to
// [PullerBatchValues] values.
func NewPuller(zr zio.Reader) Puller {
	return &puller{zr}
}

type puller struct {
	zr zio.Reader
}

func (p *puller) Pull(bool) (Batch, error) {
	if p.zr == nil {
		return nil, nil
	}
	arena := zed.NewArena()
	defer arena.Unref()
	vals := make([]zed.Value, 0, PullerBatchValues)
	for {
		val, err := p.zr.Read()
		if err != nil {
			return nil, err
		}
		if val == nil {
			p.zr = nil
			if len(vals) == 0 {
				return nil, nil
			}
			return NewBatch(arena, vals, nil, nil), nil
		}
		vals = append(vals, val.Copy(arena))
		if len(vals) >= PullerBatchValues {
			return NewBatch(arena, vals, nil, nil), nil
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

func (r *pullerReader) Read() (*zed.Value, error) {
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
