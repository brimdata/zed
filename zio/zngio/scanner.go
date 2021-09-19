package zngio

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
	"golang.org/x/sync/errgroup"
)

type scanner struct {
	ctx   context.Context
	stats zbuf.ScannerStats

	once    sync.Once
	workers []*worker

	mu        sync.Mutex
	batchChCh chan chan zbuf.Batch
	reader    *Reader

	err error // Read protected by close of batchChCh.
}

func (r *Reader) NewScanner(ctx context.Context, filter zbuf.Filter) (zbuf.Scanner, error) {
	n := runtime.GOMAXPROCS(0)
	s := &scanner{
		ctx:       ctx,
		batchChCh: make(chan chan zbuf.Batch, n),
		reader:    r,
	}
	for i := 0; i < n; i++ {
		var bf *expr.BufferFilter
		var f expr.Filter
		if filter != nil {
			var err error
			bf, err = filter.AsBufferFilter()
			if err != nil {
				return nil, err
			}
			f, err = filter.AsFilter()
			if err != nil {
				return nil, err
			}
		}
		s.workers = append(s.workers, &worker{
			bufferFilter: bf,
			filter:       f,
			scanner:      s,
		})
	}
	return s, nil
}

func (s *scanner) Pull() (zbuf.Batch, error) {
	s.once.Do(s.start)
	for {
		ch, ok := <-s.batchChCh
		if !ok {
			return nil, s.err
		}
		if batch, ok := <-ch; ok {
			return batch, nil
		}
	}
}

func (s *scanner) start() {
	g, ctx := errgroup.WithContext(s.ctx)
	for _, w := range s.workers {
		w := w
		g.Go(func() error { return w.run(ctx) })
	}
	go func() {
		s.err = g.Wait()
		close(s.batchChCh)
	}()
}

func (s *scanner) Stats() zbuf.ScannerStats {
	return zbuf.ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.stats.BytesRead),
		BytesMatched:   atomic.LoadInt64(&s.stats.BytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.stats.RecordsRead),
		RecordsMatched: atomic.LoadInt64(&s.stats.RecordsMatched),
	}
}

type worker struct {
	bufferFilter *expr.BufferFilter
	filter       expr.Filter
	scanner      *scanner
}

func (w *worker) run(ctx context.Context) error {
	var cbufCopy, recBytesCopy []byte
	for {
		ch := make(chan zbuf.Batch, 1)
		w.scanner.mu.Lock()
	again:
		// Read until we reach a compressed value message block, read a
		// record, or encounter an error.
		rec, msg, err := w.scanner.reader.readPayload(nil)
		if msg != nil {
			goto again
		}
		var format zng.CompressionFormat
		var uncompressedLen int
		var cbuf []byte
		if err == startCompressed {
			// We've reached a compressed value message block.  Read
			// it but delay decompression and scanning until after
			// we release the mutex.
			format, uncompressedLen, cbuf, err = w.scanner.reader.readCompressed()
			// cbuf, backed by the reader's buffer, must be copied
			// before we release the mutex.
			cbufCopy = copyBytes(cbufCopy, cbuf)
		} else if rec != nil {
			// We read a record.  We'll filter it after we release
			// the mutex, but we need to copy its Bytes because
			// they're backed by the reader's buffer.
			recBytesCopy = copyBytes(recBytesCopy, rec.Bytes)
			rec.Bytes = recBytesCopy
		}
		mapper := w.scanner.reader.mapper
		streamZctx := w.scanner.reader.zctx
		select {
		case w.scanner.batchChCh <- ch:
		case <-ctx.Done():
			err = ctx.Err()
		}
		w.scanner.mu.Unlock()
		if (rec == nil && uncompressedLen == 0) || err != nil {
			close(ch)
			return err
		}
		if uncompressedLen > 0 {
			// This is the fast path.  Decompress the data in
			// cbufCopy and then scan it efficiently.
			buf, err := uncompress(format, uncompressedLen, cbufCopy)
			if err != nil {
				close(ch)
				return err
			}
			batch, err := w.scanBatch(buf, mapper, streamZctx)
			if err != nil {
				close(ch)
				return err
			}
			if batch != nil {
				ch <- batch
			}
			close(ch)
			continue
		}
		// This is the slow path when data isn't compressed.  Create a
		// batch for every record that passes the filter.
		rec.Bytes = recBytesCopy
		var stats zbuf.ScannerStats
		if w.wantRecord(rec, &stats) {
			// Give rec.Bytes ownership to the new batch.
			recBytesCopy = nil
			batch := newBatch(nil)
			batch.add(rec)
			ch <- batch
		}
		w.scanner.stats.Add(stats)
		close(ch)
	}
}

func copyBytes(dst, src []byte) []byte {
	if src == nil {
		return nil
	}
	srcLen := len(src)
	if dst == nil || cap(dst) < srcLen || dst == nil {
		// If dst is nil, set it so we don't return nil.
		dst = make([]byte, srcLen)
	}
	dst = dst[:srcLen]
	copy(dst, src)
	return dst
}

func (w *worker) scanBatch(buf *buffer, mapper *resolver.Mapper, streamZctx *zson.Context) (zbuf.Batch, error) {
	// If w.bufferFilter evaluates to false, we know buf cannot contain
	// records matching w.filter.
	if w.bufferFilter != nil && !w.bufferFilter.Eval(streamZctx, buf.Bytes()) {
		atomic.AddInt64(&w.scanner.stats.BytesRead, int64(buf.length()))
		buf.free()
		return nil, nil
	}
	// Otherwise, build a batch by reading all records in the buffer.
	batch := newBatch(buf)
	var stackRec zng.Record
	var stats zbuf.ScannerStats
	for buf.length() > 0 {
		code, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		if code > zng.CtrlValueEscape {
			return nil, errors.New("zngio: control message in compressed value messaage block")
		}
		rec, err := readValue(buf, code, mapper, w.scanner.reader.validate, &stackRec)
		if err != nil {
			return nil, err
		}
		if w.wantRecord(rec, &stats) {
			batch.add(rec)
		}
	}
	w.scanner.stats.Add(stats)
	if batch.Length() == 0 {
		batch.Unref()
		return nil, nil
	}
	return batch, nil
}

func (w *worker) wantRecord(rec *zng.Record, stats *zbuf.ScannerStats) bool {
	stats.BytesRead += int64(len(rec.Bytes))
	stats.RecordsRead++
	// It's tempting to call w.bufferFilter.Eval on rec.Bytes here, but that
	// might call FieldNameFinder.Find, which could explode or return false
	// negatives because it expects a buffer of ZNG value messages, and
	// rec.Bytes is just a ZNG value.  (A ZNG value message is a header
	// indicating a type ID followed by a value of that type.)
	if w.filter == nil || w.filter(rec) {
		stats.BytesMatched += int64(len(rec.Bytes))
		stats.RecordsMatched++
		return true
	}
	return false
}
