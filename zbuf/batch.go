package zbuf

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
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
	expr.Context
	Ref()
	Unref()
	Values() []zed.Value
}

// WriteBatch writes the values in batch to zw.  If an error occurs, WriteBatch
// stops and returns the error.
func WriteBatch(zw zio.Writer, batch Batch) error {
	vals := batch.Values()
	for i := range vals {
		if err := zw.Write(&vals[i]); err != nil {
			return err
		}
	}
	return nil
}

// readBatch reads up to n records from zr and returns them as a Batch.  At EOS,
// it returns a nil or short (fewer than n records) Batch and nil error.  If an
// error is encoutered, it returns a nil Batch and the error.  Otherwise,
// readBatch returns a full Batch of n records and nil error.
func readBatch(zr zio.Reader, n int) (Batch, error) {
	recs := make([]zed.Value, 0, n)
	for len(recs) < n {
		rec, err := zr.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		// Copy the underlying buffer because the next call to
		// zr.Read may overwrite it.
		recs = append(recs, *rec.Copy())
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return NewArray(recs), nil
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

// NewPuller returns a Puller for zr that returns Batches of up to n records.
func NewPuller(zr zio.Reader, n int) Puller {
	return &puller{zr: zr, n: n}
}

type puller struct {
	zr zio.Reader
	n  int

	eos bool
}

func (p *puller) Pull(bool) (Batch, error) {
	if p.eos {
		return nil, nil
	}
	batch, err := readBatch(p.zr, p.n)
	if err == nil && (batch == nil || len(batch.Values()) < p.n) {
		p.eos = true
	}
	return batch, err
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
	idx   int
	val   zed.Value
}

func (r *pullerReader) Read() (*zed.Value, error) {
	if r.batch == nil {
		for {
			batch, err := r.p.Pull(false)
			if err != nil || batch == nil {
				return nil, err
			}
			if len(batch.Values()) == 0 {
				continue
			}
			r.batch = batch
			r.idx = 0
			break
		}
	}
	vals := r.batch.Values()
	rec := &vals[r.idx]
	r.idx++
	if r.idx == len(vals) {
		r.val.CopyFrom(rec)
		rec = &r.val
		r.batch.Unref()
		r.batch = nil
	}
	return rec, nil
}

// XXX at some point the stacked scopes should not make copies of values
// but merely refer back to the value in the wrapped batch, and we should
// ref the wrapped batch then downstream entities will unref it, but how
// do we carry the var frame through... protocol needs to be that any new
// batch created by a proc needs to preserve the var frame... we don't
// do that right now and ref counting needs to account for the dependencies.
// procs like summarize and sort that unref their input batches merely need
// to copy the first frame (of each batch) and the contract is that the
// frame will not change between multiple batches within a single-EOS event.

type batch struct {
	Batch
	vars []zed.Value
}

func NewBatch(b Batch, vals []zed.Value) Batch {
	return &batch{
		Batch: NewArray(vals),
		vars:  CopyVars(b),
	}
}

func (b *batch) Vars() []zed.Value {
	return b.vars
}

func CopyVars(b Batch) []zed.Value {
	vars := b.Vars()
	if len(vars) > 0 {
		newvars := make([]zed.Value, len(vars))
		for k, v := range vars {
			newvars[k] = *v.Copy()
		}
		vars = newvars
	}
	return vars
}
