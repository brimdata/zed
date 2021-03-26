package zngio

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// zngScanner implements scanner.Scanner.
type zngScanner struct {
	ctx          context.Context
	reader       *Reader
	bufferFilter *expr.BufferFilter
	filter       expr.Filter
	rec          zng.Record // Used to reduce memory allocations.
	span         nano.Span
	stats        zbuf.ScannerStats
}

var _ zbuf.ScannerAble = (*Reader)(nil)

// Pull implements zbuf.Scanner.Pull.
func (s *zngScanner) Pull() (zbuf.Batch, error) {
	for {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		rec, msg, err := s.reader.readPayload(&s.rec)
		if msg != nil {
			continue
		}
		if err == endBatch {
			return nil, errors.New("internal error: batch end before zngio batch started")
		}
		if err == startBatch {
			// This is the fast path.  A buffer was just decompressed
			// so we scan it efficiently here.
			batch, err := s.scanBatch()
			if err != nil {
				return nil, err
			}
			if batch == nil {
				// The entire buffer was filtered.  So move
				// on to the next chunk which may or may not
				// be compressed.
				continue
			}
			return batch, nil
		}
		if rec == nil || err != nil {
			return nil, err
		}
		// This is the slow path when data isn't compressed.
		// Create a batch for every record.  We also have to copy
		// the body of the record since it points into the peeker's
		// read buffer, so that buffer gets sent to GC.  Conceivably
		// we could make a version of the peeker that used the batch
		// and buffer pattern here, but the common case is data will
		// be compressed and we won't traverse this code path very often.
		rec, err = s.scanOne(rec)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			// XXX we don't use rec.CopyBody() here because it will
			// mark the record volatile but that zng.Record is in
			// the buffer and will get incorrectly reused.
			b := make([]byte, len(rec.Bytes))
			copy(b, rec.Bytes)
			rec.Bytes = b
			batch := newBatch(nil)
			batch.add(rec)
			return batch, nil
		}
	}
}

func (s *zngScanner) scanBatch() (zbuf.Batch, error) {
	ubuf := s.reader.uncompressedBuf
	// If s.bufferFilter evaluates to false, we know ubuf cannot
	// contain records matching s.filter.
	if s.bufferFilter != nil && !s.bufferFilter.Eval(s.reader.zctx, ubuf.Bytes()) {
		atomic.AddInt64(&s.stats.BytesRead, int64(ubuf.length()))
		ubuf.free()
		s.reader.uncompressedBuf = nil
		return nil, nil
	}
	// Otherwise, build a batch by reading all records in the
	// decompressed buffer (i.e., till endBatch).
	batch := newBatch(ubuf)
	for {
		rec, msg, err := s.reader.readPayload(&s.rec)
		if msg != nil {
			continue
		}
		if err == endBatch {
			if batch.Length() == 0 {
				batch.Unref()
				return nil, nil
			}
			return batch, nil
		}
		if err != nil {
			return nil, err
		}

		if rec == nil {
			// this shouldn't happen
			return nil, errors.New("zngio: null record in middle of compressed batch")
		}
		rec, err = s.scanOne(rec)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			batch.add(rec)
		}
	}
}

func (s *zngScanner) scanOne(rec *zng.Record) (*zng.Record, error) {
	atomic.AddInt64(&s.stats.BytesRead, int64(len(rec.Bytes)))
	atomic.AddInt64(&s.stats.RecordsRead, 1)
	if s.span != nano.MaxSpan && !s.span.Contains(rec.Ts()) ||
		s.filter != nil && !s.filter(rec) {
		return nil, nil
	}
	atomic.AddInt64(&s.stats.BytesMatched, int64(len(rec.Bytes)))
	atomic.AddInt64(&s.stats.RecordsMatched, 1)
	return rec, nil
}

// Stats implements zbuf.Scanner.Stats.
func (s *zngScanner) Stats() *zbuf.ScannerStats {
	return &zbuf.ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.stats.BytesRead),
		BytesMatched:   atomic.LoadInt64(&s.stats.BytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.stats.RecordsRead),
		RecordsMatched: atomic.LoadInt64(&s.stats.RecordsMatched),
	}
}
