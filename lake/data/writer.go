package data

import (
	"bytes"
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
)

// Writer is a zio.Writer that writes a stream of sorted records into a
// data object.
type Writer struct {
	object           *Object
	byteCounter      *writeCounter
	count            uint64
	writer           *zngio.Writer
	lastSOS          int64
	order            order.Which
	seekWriter       *zngio.Writer
	seekIndex        *seekindex.Writer
	seekIndexStride  int
	seekIndexTrigger int
	first            bool
	poolKey          field.Path
}

// NewWriter returns a writer for writing the data of a zng-row storage object as
// well as optionally creating a seek index for the row object when the
// seekIndexStride is non-zero.  We assume all records are non-volatile until
// Close as zed.Values from the various record bodies are referenced across
// calls to Write.
func (o *Object) NewWriter(ctx context.Context, engine storage.Engine, path *storage.URI, order order.Which, poolKey field.Path, seekIndexStride int) (*Writer, error) {
	out, err := engine.Put(ctx, o.SequenceURI(path))
	if err != nil {
		return nil, err
	}
	counter := &writeCounter{bufwriter.New(out), 0}
	w := &Writer{
		object:      o,
		byteCounter: counter,
		writer:      zngio.NewWriter(counter),
		order:       order,
		first:       true,
		poolKey:     poolKey,
	}
	if seekIndexStride == 0 {
		seekIndexStride = DefaultSeekStride
	}
	w.seekIndexStride = seekIndexStride
	seekOut, err := engine.Put(ctx, o.SeekIndexURI(path))
	if err != nil {
		return nil, err
	}
	w.seekWriter = zngio.NewWriter(bufwriter.New(seekOut))
	w.seekIndex = seekindex.NewWriter(w.seekWriter)
	return w, nil
}

func (w *Writer) Write(rec *zed.Value) error {
	key := rec.DerefPath(w.poolKey).MissingAsNull()
	if w.seekIndex != nil {
		if err := w.writeIndex(*key); err != nil {
			return err
		}
	}
	if err := w.writer.Write(rec); err != nil {
		return err
	}
	w.object.Last.CopyFrom(key)
	w.count++
	return nil
}

func (w *Writer) writeIndex(key zed.Value) error {
	w.seekIndexTrigger += len(key.Bytes)
	if w.first {
		w.first = false
		w.object.First.CopyFrom(&key)
		w.object.Last.CopyFrom(&key)
		return w.seekIndex.Write(key, 0, 0)
	}
	if w.seekIndexTrigger < w.seekIndexStride || bytes.Equal(key.Bytes, w.object.Last.Bytes) {
		return nil
	}
	if err := w.writer.EndStream(); err != nil {
		return err
	}
	w.seekIndexTrigger = 0
	pos := w.writer.Position()
	return w.seekIndex.Write(key, w.count, pos)
}

// Abort is called when an error occurs during write. Errors are ignored
// because the write error will be more informative and should be returned.
func (w *Writer) Abort() {
	w.writer.Close()
	w.seekWriter.Close()
}

func (w *Writer) Close(ctx context.Context) error {
	err := w.writer.Close()
	if err != nil {
		w.Abort()
		return err
	}
	if err := w.seekWriter.Close(); err != nil {
		w.Abort()
		return err
	}
	w.object.Count = w.count
	w.object.Size = w.writer.Position()
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
