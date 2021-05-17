package segment

import (
	"bytes"
	"context"
	"io"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
)

// Writer is a zbuf.Writer that writes a stream of sorted records into a
// data segment.
type Writer struct {
	ref             *Reference
	byteCounter     *writeCounter
	count           uint64
	rowObject       *zngio.Writer
	firstKey        zng.Value
	lastKey         zng.Value
	needSeekWrite   bool
	order           order.Which
	seekIndex       *seekindex.Writer
	seekIndexCloser io.Closer
	first           bool
	poolKey         field.Path
}

func (r *Reference) NewWriter(ctx context.Context, engine storage.Engine, path *storage.URI, o order.Which, seekIndexFactor int, poolKey field.Path) (*Writer, error) {
	out, err := engine.Put(ctx, r.RowObjectPath(path))
	if err != nil {
		return nil, err
	}
	counter := &writeCounter{bufwriter.New(out), 0}
	opts := zngio.WriterOpts{
		StreamRecordsMax: seekIndexFactor,
		LZ4BlockSize:     zngio.DefaultLZ4BlockSize,
	}
	writer := zngio.NewWriter(counter, opts)
	seekOut, err := engine.Put(ctx, r.SeekObjectPath(path))
	if err != nil {
		return nil, err
	}
	opts = zngio.WriterOpts{
		//LZ4BlockSize: zngio.DefaultLZ4BlockSize,
	}
	seekWriter := zngio.NewWriter(bufwriter.New(seekOut), opts)
	return &Writer{
		ref:             r,
		byteCounter:     counter,
		rowObject:       writer,
		seekIndex:       seekindex.NewWriter(seekWriter),
		seekIndexCloser: seekWriter,
		order:           o,
		first:           true,
		poolKey:         poolKey,
	}, nil
}

func (w *Writer) Write(rec *zng.Record) error {
	// We want to index the start of stream (SOS) position of the data file by
	// record timestamps; we don't know when we've started a new stream until
	// after we have written the first record in the stream.
	sos := w.rowObject.LastSOS()
	if err := w.rowObject.Write(rec); err != nil {
		return err
	}
	key, err := rec.Deref(w.poolKey)
	if err != nil {
		key = zng.Value{zng.TypeNull, nil}
	}
	if w.first {
		w.first = false
		w.firstKey = key
		if err := w.seekIndex.Write(key, sos); err != nil {
			return err
		}
	} else if w.needSeekWrite && (w.lastKey.Bytes == nil || !bytes.Equal(key.Bytes, w.lastKey.Bytes)) {
		if err := w.seekIndex.Write(key, sos); err != nil {
			return err
		}
		w.needSeekWrite = false
	}
	if w.rowObject.LastSOS() != sos {
		w.needSeekWrite = true
	}
	w.lastKey = key
	w.count++
	return nil
}

// Abort is called when an error occurs during write. Errors are ignored
// because the write error will be more informative and should be returned.
func (w *Writer) Abort() {
	w.rowObject.Close()
	w.seekIndexCloser.Close()
}

func (w *Writer) Close(ctx context.Context) error {
	err := w.rowObject.Close()
	if err != nil {
		w.Abort()
		return err
	}
	if err := w.seekIndexCloser.Close(); err != nil {
		w.Abort()
		return err
	}
	w.ref.Count = w.count
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
