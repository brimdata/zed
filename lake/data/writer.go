package data

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

// Writer is a zio.Writer that writes a stream of sorted records into a
// data object.
type Writer struct {
	object           *Object
	byteCounter      *writeCounter
	count            uint64
	rowObject        *zngio.Writer
	firstKey         zng.Value
	lastKey          zng.Value
	lastSOS          int64
	order            order.Which
	seekIndex        *seekindex.Writer
	seekIndexCloser  io.Closer
	seekIndexStride  int
	seekIndexTrigger int
	first            bool
	poolKey          field.Path
}

// NewWriter returns a writer for writing the data of a zng-row storage object as
// well as optionally creating a seek index for the row object when the
// seekIndexStride is non-zero.  We assume all records are non-volatile until
// Close as zng.Values from the various record bodies are referenced across
// calls to Write.
func (o *Object) NewWriter(ctx context.Context, engine storage.Engine, path *storage.URI, order order.Which, poolKey field.Path, seekIndexStride int) (*Writer, error) {
	out, err := engine.Put(ctx, o.RowObjectPath(path))
	if err != nil {
		return nil, err
	}
	counter := &writeCounter{bufwriter.New(out), 0}
	writer := zngio.NewWriter(counter, zngio.WriterOpts{
		LZ4BlockSize: zngio.DefaultLZ4BlockSize,
	})
	w := &Writer{
		object:      o,
		byteCounter: counter,
		rowObject:   writer,
		order:       order,
		first:       true,
		poolKey:     poolKey,
	}
	if seekIndexStride != 0 {
		w.seekIndexStride = seekIndexStride
		seekOut, err := engine.Put(ctx, o.SeekObjectPath(path))
		if err != nil {
			return nil, err
		}
		opts := zngio.WriterOpts{
			//LZ4BlockSize: zngio.DefaultLZ4BlockSize,
		}
		seekWriter := zngio.NewWriter(bufwriter.New(seekOut), opts)
		w.seekIndex = seekindex.NewWriter(seekWriter)
		w.seekIndexCloser = seekWriter
	}
	return w, nil
}

func (w *Writer) Write(rec *zng.Record) error {
	key, err := rec.Deref(w.poolKey)
	if err != nil {
		key = zng.Value{zng.TypeNull, nil}
	}
	if w.seekIndex != nil {
		if err := w.writeIndex(key); err != nil {
			return err
		}
	}
	if err := w.rowObject.Write(rec); err != nil {
		return err
	}
	w.lastKey = key
	w.count++
	return nil
}

func (w *Writer) writeIndex(key zng.Value) error {
	w.seekIndexTrigger += len(key.Bytes)
	if w.first {
		w.first = false
		w.firstKey = key
		w.lastKey = key
		return w.seekIndex.Write(key, 0)
	}
	if w.seekIndexTrigger < w.seekIndexStride || bytes.Equal(key.Bytes, w.lastKey.Bytes) {
		return nil
	}
	if err := w.rowObject.EndStream(); err != nil {
		return err
	}
	pos := w.rowObject.Position()
	if err := w.seekIndex.Write(key, pos); err != nil {
		return err
	}
	w.seekIndexTrigger = 0
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
	w.object.Count = w.count
	w.object.RowSize = w.rowObject.Position()
	return nil
}

func (w *Writer) BytesWritten() int64 {
	return w.byteCounter.size
}

func (w *Writer) RecordsWritten() uint64 {
	return w.count
}

// Object returns the Object written by the writer. This is only valid after
// Close() has returned a nil error.
func (w *Writer) Object() *Object {
	return w.object
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
