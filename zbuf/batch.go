package zbuf

import (
	"github.com/brimsec/zq/zng"
)

// Batch is an interface to a bundle of records.
// Batches can be shared across goroutines via reference counting and should be
// copied on modification when the reference count is greater than 1.
type Batch interface {
	Ref()
	Unref()
	Index(int) *zng.Record
	Length() int
	Records() []*zng.Record
}

// ReadBatch reads up to n records read from zr and returns them as a Batch.  At
// EOF, it returns a nil Batch and nil error.  If an error is encoutered, it
// returns a nil Batch and the error.
func ReadBatch(zr Reader, n int) (Batch, error) {
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
		rec.CopyBody()
		recs = append(recs, rec)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return NewArray(recs), nil
}

// A Puller produces Batches of records, signaling end-of-stream by returning
// a nil Batch and nil error.
type Puller interface {
	Pull() (Batch, error)
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
