package lake

import (
	"context"
	"sync/atomic"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"golang.org/x/sync/errgroup"
)

// Writer is a zio.Writer that consumes records into memory according to
// the pools data object threshold, sorts each resulting buffer, and writes
// it as an immutable object to the storage system.  The presumption is that
// each buffer's worth of data fits into memory.
type Writer struct {
	pool        *Pool
	objects     []data.Object
	inputSorted bool
	ctx         context.Context
	zctx        *zed.Context
	//defs          index.Definitions
	errgroup *errgroup.Group
	vals     []zed.Value
	// XXX this is a simple double buffering model so the cloud-object
	// writer can run in parallel with the reader filling the records
	// buffer.  This can be later extended to pass a big bytes buffer
	// back and forth where the bytes buffer holds all of the record
	// data efficiently in one big backing store.
	buffer      chan []zed.Value
	comparator  *expr.Comparator
	sorter      expr.Sorter
	memBuffered int64
	stats       ImportStats
}

//XXX NOTE: we removed the flusher logic as the callee should just put
// a timeout on the context.  We will catch that timeout here and push
// all records that have been consumed and return the commits of everything
// that made it up to the timeout.  This provides a mechanism for streaming
// microbatches with a timeout defined from above and a nice way to sync the
// timeout with the commit rather than trying to do all of this bottoms up.

// NewWriter creates a zio.Writer compliant writer for writing data to an
// a data pool presuming the input is not guaranteed to be sorted.
//XXX we should make another writer that takes sorted input and is a bit
// more efficient.  This other writer could have different commit triggers
// to do useful things like paritioning given the context is a rollup.
func NewWriter(ctx context.Context, zctx *zed.Context, pool *Pool) (*Writer, error) {
	g, ctx := errgroup.WithContext(ctx)
	ch := make(chan []zed.Value, 1)
	ch <- nil
	return &Writer{
		pool:       pool,
		ctx:        ctx,
		zctx:       zctx,
		errgroup:   g,
		buffer:     ch,
		comparator: importComparator(zctx, pool),
	}, nil
}

func (w *Writer) Objects() []data.Object {
	return w.objects
}

func (w *Writer) newObject() *data.Object {
	w.objects = append(w.objects, data.NewObject())
	return &w.objects[len(w.objects)-1]
}

func (w *Writer) Write(rec *zed.Value) error {
	if w.ctx.Err() != nil {
		if err := w.errgroup.Wait(); err != nil {
			return err
		}
		return w.ctx.Err()
	}
	w.append(rec)
	w.memBuffered += int64(len(rec.Bytes))
	//XXX change name LogSizeThreshold
	// XXX the previous logic estimated the object size with divide by 2...?!
	if w.memBuffered >= w.pool.Threshold {
		w.flipBuffers()
	}
	return nil
}

func (w *Writer) append(val *zed.Value) {
	n := len(w.vals)
	if n < cap(w.vals) {
		w.vals = w.vals[:n+1]
	} else {
		w.vals = append(w.vals, zed.Value{})
	}
	w.vals[n].CopyFrom(val)
}

func (w *Writer) flipBuffers() {
	oldvals, ok := <-w.buffer
	if !ok {
		return
	}
	recs := w.vals
	w.vals = oldvals[:0]
	w.memBuffered = 0
	w.errgroup.Go(func() error {
		err := w.writeObject(w.newObject(), recs)
		if err != nil {
			close(w.buffer)
			return err
		}
		w.buffer <- recs
		return err
	})
}

func (w *Writer) Close() error {
	// Send the last write (Note: we could reorder things so we do the
	// record sort in this thread while waiting for the write to complete.)
	if len(w.vals) > 0 {
		w.flipBuffers()
	}
	// Wait for any pending write to finish.
	return w.errgroup.Wait()
}

func (w *Writer) writeObject(object *data.Object, recs []zed.Value) error {
	if !w.inputSorted {
		done := make(chan struct{})
		go func() {
			w.sorter.SortStable(recs, w.comparator)
			close(done)
		}()
		select {
		case <-done:
		case <-w.ctx.Done():
			return w.ctx.Err()
		}
	}
	// Set first and last key values after the sort.
	key := poolKey(w.pool.Layout)
	object.First = *recs[0].DerefPath(key).MissingAsNull()
	object.Last = *recs[len(recs)-1].DerefPath(key).MissingAsNull()
	writer, err := object.NewWriter(w.ctx, w.pool.engine, w.pool.DataPath, w.pool.Layout.Order, key, w.pool.SeekStride)
	if err != nil {
		return err
	}
	r := zbuf.NewArray(recs).NewReader()
	if err := zio.CopyWithContext(w.ctx, writer, r); err != nil {
		writer.Abort()
		return err
	}
	if err := writer.Close(w.ctx); err != nil {
		return err
	}
	w.stats.Accumulate(ImportStats{
		ObjectsWritten:     1,
		RecordBytesWritten: writer.BytesWritten(),
		RecordsWritten:     int64(writer.RecordsWritten()),
	})
	return nil
}

func (w *Writer) Stats() ImportStats {
	return w.stats.Copy()
}

type ImportStats struct {
	ObjectsWritten     int64
	RecordBytesWritten int64
	RecordsWritten     int64
}

func (s *ImportStats) Accumulate(b ImportStats) {
	atomic.AddInt64(&s.ObjectsWritten, b.ObjectsWritten)
	atomic.AddInt64(&s.RecordBytesWritten, b.RecordBytesWritten)
	atomic.AddInt64(&s.RecordsWritten, b.RecordsWritten)
}

func (s *ImportStats) Copy() ImportStats {
	return ImportStats{
		ObjectsWritten:     atomic.LoadInt64(&s.ObjectsWritten),
		RecordBytesWritten: atomic.LoadInt64(&s.RecordBytesWritten),
		RecordsWritten:     atomic.LoadInt64(&s.RecordsWritten),
	}
}

func importComparator(zctx *zed.Context, pool *Pool) *expr.Comparator {
	layout := pool.Layout
	layout.Keys = field.List{poolKey(layout)}
	return zbuf.NewComparator(zctx, layout)
}

func poolKey(layout order.Layout) field.Path {
	if len(layout.Keys) != 0 {
		// XXX We don't yet handle multiple pool keys.
		return layout.Keys[0]
	}
	return field.New("ts")
}
