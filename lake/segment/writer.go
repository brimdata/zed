package segment

import (
	"context"
	"io"

	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
)

// Writer is a zbuf.Writer that writes a stream of sorted records into a
// data segment.
type Writer struct {
	ref           *Reference
	byteCounter   *writeCounter
	count         uint64
	size          int64
	rowObject     *zngio.Writer
	firstTs       nano.Ts
	lastTs        nano.Ts
	needSeekWrite bool
	order         order.Which
	seekIndex     *seekindex.Builder
	wroteFirst    bool
}

type WriterOpts struct {
	Order order.Which
	Zng   zngio.WriterOpts
}

func (r *Reference) NewWriter(ctx context.Context, engine storage.Engine, path *storage.URI, opts WriterOpts) (*Writer, error) {
	out, err := engine.Put(ctx, r.RowObjectPath(path))
	if err != nil {
		return nil, err
	}
	counter := &writeCounter{bufwriter.New(out), 0}
	writer := zngio.NewWriter(counter, opts.Zng)
	seekObjectPath := r.SeekObjectPath(path)
	seekIndex, err := seekindex.NewBuilder(ctx, engine, seekObjectPath.String(), opts.Order)
	if err != nil {
		return nil, err
	}
	return &Writer{
		ref:         r,
		byteCounter: counter,
		rowObject:   writer,
		seekIndex:   seekIndex,
		order:       opts.Order,
	}, nil
}

type indexWriter interface {
	zio.WriteCloser
	Abort()
}

type nopIndexWriter struct{}

func (nopIndexWriter) Write(*zng.Record) error { return nil }
func (nopIndexWriter) Close() error            { return nil }
func (nopIndexWriter) Abort()                  {}

func (w *Writer) Position() (int64, nano.Ts, nano.Ts) {
	return w.rowObject.Position(), w.firstTs, w.lastTs
}

func (w *Writer) Write(rec *zng.Record) error {
	// We want to index the start of stream (SOS) position of the data file by
	// record timestamps; we don't know when we've started a new stream until
	// after we have written the first record in the stream.
	sos := w.rowObject.LastSOS()
	if err := w.rowObject.Write(rec); err != nil {
		return err
	}
	//XXX this is a complicated way to avoid splitting zng frames in the middle of
	// the same key.  we should call EndStream explicitly instead of setting
	// the number-of-records trigger.  See issue #XXX.
	ts := rec.Ts()
	if !w.wroteFirst || (w.needSeekWrite && ts != w.lastTs) {
		if err := w.seekIndex.Enter(ts, sos); err != nil {
			return err
		}
		w.needSeekWrite = false
	}
	if w.rowObject.LastSOS() != sos {
		w.needSeekWrite = true
	}
	if !w.wroteFirst {
		w.firstTs = ts
		w.wroteFirst = true
	}
	w.lastTs = ts
	w.count++
	w.size += int64(len(rec.Bytes))
	return nil
}

// Abort is called when an error occurs during write. Errors are ignored
// because the write error will be more informative and should be returned.
func (w *Writer) Abort() {
	w.rowObject.Close()
	w.seekIndex.Abort()
}

func (w *Writer) Close(ctx context.Context) error {
	err := w.rowObject.Close()
	if err != nil {
		w.Abort()
		return err
	}
	if err := w.seekIndex.Close(); err != nil {
		w.Abort()
		return err
	}
	w.ref.Count = w.count
	w.ref.Size = w.size
	w.ref.RowSize = w.rowObject.Position()
	return nil
}

func (w *Writer) BytesWritten() int64 {
	return w.byteCounter.size
}

func (w *Writer) RecordsWritten() uint64 {
	return w.count
}

// Segment returns the Segment written by the writer. This is only valid after
// Close() has returned a nil error.
func (w *Writer) Segment() *Reference {
	return w.ref
}

type writeCounter struct {
	io.WriteCloser
	size int64
}

func (w *writeCounter) Write(b []byte) (int, error) {
	n, err := w.WriteCloser.Write(b)
	w.size += int64(n)
	return n, err
}
