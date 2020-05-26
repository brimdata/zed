package zbuf

import (
	"github.com/brimsec/zq/pkg/nano"
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
	//XXX span should go in here?
	Span() nano.Span
}

// ReadBatch reads up to n records read from zr and returns them as a Batch.  At
// EOF, it returns a nil Batch and nil error.  If an error is encoutered, it
// returns a nil Batch and the error.
func ReadBatch(zr Reader, n int) (Batch, error) {
	minTs, maxTs := nano.MaxTs, nano.MinTs
	recs := make([]*zng.Record, 0, n)
	for len(recs) < n {
		rec, err := zr.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if rec.Ts < minTs {
			minTs = rec.Ts
		}
		if rec.Ts > maxTs {
			maxTs = rec.Ts
		}
		// Copy the underlying buffer (if volatile) because call to next
		// reader.Next() may overwrite said buffer.
		rec.CopyBody()
		recs = append(recs, rec)
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return NewArray(recs, nano.NewSpanTs(minTs, maxTs)), nil
}

func ReadBatchSize(zr Reader, n int64) (Batch, error) {
	minTs, maxTs := nano.MaxTs, nano.MinTs
	var size int64
	recs := make([]*zng.Record, 0, n)
	for size < n {
		rec, err := zr.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if rec.Ts < minTs {
			minTs = rec.Ts
		}
		if rec.Ts > maxTs {
			maxTs = rec.Ts
		}
		// Copy the underlying buffer (if volatile) because call to next
		// reader.Next() may overwrite said buffer.
		rec.CopyBody()
		recs = append(recs, rec)
		size += int64(len(rec.Raw))
	}
	if len(recs) == 0 {
		return nil, nil
	}
	return NewArray(recs, nano.NewSpanTs(minTs, maxTs)), nil

}

type batchReader struct {
	batch Batch
	n     int
}

func NewBatchReader(batch Batch) Reader {
	return &batchReader{batch: batch}
}

func (b *batchReader) Read() (*zng.Record, error) {
	if b.n >= b.batch.Length() {
		return nil, nil
	}
	rec := b.batch.Index(b.n)
	b.n++
	return rec, nil
}
