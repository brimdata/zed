package lake

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/proc/sort"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"golang.org/x/sync/errgroup"
)

var (
	// ImportBufSize specifies the max size of the records buffered during import
	// before they are flushed to disk.
	ImportBufSize          = int64(sort.MemMaxBytes)
	ImportStreamRecordsMax = zngio.DefaultStreamRecordsMax

	// For unit testing.
	importLZ4BlockSize = zngio.DefaultLZ4BlockSize
)

const defaultCommitTimeout = time.Second * 5

// Writer is a zbuf.Writer that partitions records by day into the
// appropriate tsDirWriter. Writer keeps track of the overall memory
// footprint of the collection of tsDirWriter and instructs the tsDirWriter
// with the largest footprint to spill its records to a temporary file on disk.
//
// TODO issue 1432 Writer does not currently keep track of size of records
// written to temporary files. At some point this should have a maxTempFileSize
// to ensure the Writer does not exceed the size of a provisioned tmpfs.
//
// XXX When the expected size of writing the records is greater than
// lk.LogSizeThreshold, they are written to a chunk file in
// the archive.
type Writer struct {
	pool        *Pool
	segments    []segment.Reference
	inputSorted bool
	ctx         context.Context
	//defs          index.Definitions
	errgroup *errgroup.Group
	records  []*zng.Record
	// XXX this is a simple double buffering model so the cloud-object
	// writer can run in parallel with the reader filling the records
	// buffer.  This can be later extended to pass a big bytes buffer
	// back and forth where the bytes buffer holds all of the record
	// data efficiently in one big backing store.
	buffer chan []*zng.Record

	memBuffered int64
	stats       ImportStats
}

//XXX NOTE: we removed the flusher logic as the callee should just put
// a timeout on the context.  We will catch that timeout here and push
// all records that have been consumed and return the commits of everything
// that made it up to the timeout.  This provides a mechanism for streaming
// microbatches with a timeout defined from above and a nice way to sync the
// timeout with the commit rather than trying to do all of this bottoms up.

// NewWriter creates a zbuf.Writer compliant writer for writing data to an
// a data pool presuming the input is not guaranteed to be sorted.
//XXX we should make another writer that takes sorted input and is a bit
// more efficient.  This other writer could have different commit triggers
// to do useful things like paritioning given the context is a rollup.
func NewWriter(ctx context.Context, pool *Pool) (*Writer, error) {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan []*zng.Record, 1)
	ch <- nil
	w := &Writer{
		pool:     pool,
		ctx:      ctx,
		errgroup: g,
		buffer:   ch,
	}
	return w, nil
}

func (w *Writer) Segments() []segment.Reference {
	return w.segments
}

func (w *Writer) newSegment() *segment.Reference {
	w.segments = append(w.segments, segment.New())
	return &w.segments[len(w.segments)-1]
}

func (w *Writer) Write(rec *zng.Record) error {
	if w.ctx.Err() != nil {
		if err := w.errgroup.Wait(); err != nil {
			return err
		}
		return w.ctx.Err()
	}
	// XXX This call leads to a ton of one-off allocations that burden the GC
	// and slow down import. We should instead copy the raw record bytes into a
	// recycled buffer and keep around an array of ts + byte-slice structs for
	// sorting.
	rec.CopyBytes()
	w.records = append(w.records, rec)
	w.memBuffered += int64(len(rec.Bytes))
	//XXX change name LogSizeThreshold
	// XXX the previous logic estimated the segment size with divide by 2...?!
	if w.memBuffered >= w.pool.Treshold {
		w.flipBuffers()
	}
	return nil
}

func (w *Writer) flipBuffers() {
	oldrecs := <-w.buffer
	recs := w.records
	w.records = oldrecs[:0]
	w.memBuffered = 0
	seg := w.newSegment()
	w.errgroup.Go(func() error {
		err := w.writeObject(seg, recs)
		w.buffer <- recs
		return err
	})
}

func (w *Writer) Close() error {
	// Send the last write (Note: we could reorder things so we do the
	// record sort in this thread while waiting for the write to complete.)
	if len(w.records) > 0 {
		w.flipBuffers()
	}
	// Wait for any pending write to finish.
	err := w.errgroup.Wait()
	if err != nil {
		return err
	}
	return err
}

func (w *Writer) writeObject(seg *segment.Reference, recs []*zng.Record) error {
	if !w.inputSorted {
		expr.SortStable(recs, importCompareFn(w.pool))
	}
	// Set first and last key values after the sort.
	seg.First = recs[0].Ts()
	seg.Last = recs[len(recs)-1].Ts()
	r := zbuf.Array(recs).NewReader()
	writer, err := seg.NewWriter(w.ctx, w.pool.DataPath, segment.WriterOpts{
		Order: w.pool.Order,
		Zng: zngio.WriterOpts{
			StreamRecordsMax: ImportStreamRecordsMax,
			LZ4BlockSize:     importLZ4BlockSize,
		},
	})
	if err != nil {
		return err
	}
	if err := zbuf.CopyWithContext(w.ctx, writer, r); err != nil {
		writer.Abort()
		return err
	}
	if err := writer.Close(w.ctx); err != nil {
		return err
	}
	w.stats.Accumulate(ImportStats{
		DataChunksWritten:  1,
		RecordBytesWritten: writer.BytesWritten(),
		RecordsWritten:     int64(writer.RecordsWritten()),
	})
	return nil
}

func (w *Writer) Stats() ImportStats {
	return w.stats.Copy()
}

type ImportStats struct {
	DataChunksWritten  int64
	RecordBytesWritten int64
	RecordsWritten     int64
}

func (s *ImportStats) Accumulate(b ImportStats) {
	atomic.AddInt64(&s.DataChunksWritten, b.DataChunksWritten)
	atomic.AddInt64(&s.RecordBytesWritten, b.RecordBytesWritten)
	atomic.AddInt64(&s.RecordsWritten, b.RecordsWritten)
}

func (s *ImportStats) Copy() ImportStats {
	return ImportStats{
		DataChunksWritten:  atomic.LoadInt64(&s.DataChunksWritten),
		RecordBytesWritten: atomic.LoadInt64(&s.RecordBytesWritten),
		RecordsWritten:     atomic.LoadInt64(&s.RecordsWritten),
	}
}

func importCompareFn(pool *Pool) expr.CompareFn {
	return zbuf.NewCompareFn(field.New("ts"), pool.Order == zbuf.OrderDesc)
}
