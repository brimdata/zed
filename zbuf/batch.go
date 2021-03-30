package zbuf

import (
	"github.com/brimdata/zed/zng"
)

// Batch is an interface to a bundle of records.  Reference counting allows
// efficient, safe reuse in concert with sharing across goroutines.
//
// A newly obtained Batch always has a reference count of one.  The Batch owns
// its records and their storage, and an implementation may reuse this memory
// when the reference count falls to zero, reducing load on the garbage
// collector.
//
// To promote reuse, a goroutine should release a Batch reference when possible,
// but care must be taken to avoid race conditions that arise from releasing a
// reference too soon.  Specifically, a goroutine obtaining a *zng.Record from a
// Batch must retain its Batch reference for as long as it retains the pointer,
// and the goroutine must not use the pointer after releasing its reference.
//
// Regardless of reference count or implementation, an unreachable Batch will
// eventually be reclaimed by the garbage collector.
type Batch interface {
	Ref()
	Unref()
	Index(int) *zng.Record
	Length() int
	Records() []*zng.Record
}

// readBatch reads up to n records from zr and returns them as a Batch.  At EOS,
// it returns a nil or short (fewer than n records) Batch and nil error.  If an
// error is encoutered, it returns a nil Batch and the error.  Otherwise,
// readBatch returns a full Batch of n records and nil error.
func readBatch(zr Reader, n int) (Batch, error) {
	recs := make([]*zng.Record, 0, n)
	for len(recs) < n {
		rec, err := zr.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		// Copy the underlying buffer (if volatile) because call to next
		// reader.Next() may overwrite said buffer.
		rec.CopyBytes()
		recs = append(recs, rec)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return Array(recs), nil
}

// A Puller produces Batches of records, signaling end-of-stream by returning
// a nil Batch and nil error.
type Puller interface {
	Pull() (Batch, error)
}

// NewPuller returns a Puller for zr that returns Batches of up to n records.
func NewPuller(zr Reader, n int) Puller {
	return &puller{zr: zr, n: n}
}

type puller struct {
	zr Reader
	n  int

	eos bool
}

func (p *puller) Pull() (Batch, error) {
	if p.eos {
		return nil, nil
	}
	batch, err := readBatch(p.zr, p.n)
	if err == nil && (batch == nil || batch.Length() < p.n) {
		p.eos = true
	}
	return batch, err
}

func CopyPuller(w Writer, p Puller) error {
	for {
		b, err := p.Pull()
		if b == nil || err != nil {
			return err
		}
		for _, r := range b.Records() {
			if err := w.Write(r); err != nil {
				return err
			}
		}
		b.Unref()
	}
}

func PullerReader(p Puller) Reader {
	return &pullerReader{p: p}
}

type pullerReader struct {
	p     Puller
	batch Batch
	idx   int
}

func (r *pullerReader) Read() (*zng.Record, error) {
	if r.batch == nil {
		for {
			batch, err := r.p.Pull()
			if err != nil || batch == nil {
				return nil, err
			}
			if batch.Length() == 0 {
				continue
			}
			r.batch = batch
			r.idx = 0
			break
		}
	}
	rec := r.batch.Index(r.idx)
	r.idx++
	if r.idx == r.batch.Length() {
		r.batch = nil
	}
	return rec, nil
}
