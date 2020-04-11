package zdx

import (
	"errors"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

//XXX careful about mismatch between writer thesh and reader frame size causing
// a bit of a perf hit

// Writer returns a zng writer that creates a zdx bundle,
// comprising the base zng file along with its related b-tree files,
// as zng records are consumed.  The records must all have the same
// type and be comprised of two columns where the first column name is
// "key" and the second column name is "value."
//
// A zdx base file is a zng file represented as sequence of zng records where
// the records have either a single column named "key" or two columns named
// "key" and "value".  The records are sorted by the key field.
// Once the zng file data is written, the b-tree index files comprise a
// constant b-tree to make key lookups efficient.  The b-tree files
// are zng files where each record is comprised of a "key" field in
// column 0 and an "offset" field in column 1.  The key field is an arbitrary
// zng type while the offset field must be a zng int64.  The offset field
// corresponds to the seek offset of the b-tree file next below it in the
// hierarchy where that key is found.
type Writer struct {
	path        string
	level       int
	writer      *bufwriter.Writer
	out         *bzngio.Writer
	parent      *Writer
	frameThresh int
	frameStart  int64
	frameEnd    int64
	frameKey    *zng.Value
	keyBuf      zng.Value
	builder     *zng.Builder
	recType     *zng.TypeRecord
}

// NewWriter returns a Writer ready to write an zdx bundle and related
// index files via subsequent calls to Write(), or it returns an error.
// All files will be written to the directory indicated by path
// with the form $path, $path.1, and so forth.  Calls to Write must
// provide keys in increasing lexicographic order.  Duplicate keys are not
// allowed but will not be detected.  Close() must be called when done writing.
func NewWriter(path string, framesize int) (*Writer, error) {
	return newWriter(path, framesize, 0)
}

func newWriter(path string, framesize, level int) (*Writer, error) {
	if level > 5 {
		panic("something wrong")
	}
	name := filename(path, level)
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	writer := bufwriter.New(f)
	return &Writer{
		path:        path,
		level:       level,
		writer:      writer,
		out:         bzngio.NewWriter(writer, zio.Flags{}),
		frameThresh: framesize,
		frameEnd:    int64(framesize),
	}, nil
}

// Flush is here to implement zbuf.WriteFluser
func (w *Writer) Flush() error {
	return w.writer.Flush()
}

func (w *Writer) Close() error {
	// Make sure to pass up framekeys to parent trees, even though frames aren't
	// full.
	if w.parent != nil && w.frameKey != nil {
		if err := w.endFrame(); err != nil {
			return err
		}
	}
	if err := w.writer.Close(); err != nil {
		return err
	}
	if w.parent != nil {
		return w.parent.Close()
	}
	return nil
}

func (w *Writer) Write(rec *zng.Record) error {
	offset := w.out.Position()
	if offset >= w.frameEnd {
		w.frameEnd = offset + int64(w.frameThresh)
		// the frame in place is already big enough... flush it and
		// start going on the next
		if err := w.endFrame(); err != nil {
			return err
		}
	}
	// Remember the first key of a new frame. This happens at beginning
	// of stream or when we end the current frame immediately above.
	if w.frameKey == nil {
		key := rec.Value(0)
		if key.Type == nil {
			return ErrCorruptFile
		}
		// Copy the key value from the stream so we can write it to the
		// parent when we hit end-of-frame.
		w.keyBuf.Type = key.Type
		w.keyBuf.Bytes = append(w.keyBuf.Bytes[:0], key.Bytes...)
		w.frameKey = &w.keyBuf
	}
	return w.out.Write(rec)
}

func (w *Writer) endFrame() error {
	if err := w.addToParentIndex(w.frameKey, w.frameStart); err != nil {
		return err
	}
	w.out.EndStream()
	w.frameStart = w.out.Position()
	w.frameKey = nil
	return nil
}

func (w *Writer) addToParentIndex(key *zng.Value, offset int64) error {
	if w.parent == nil {
		var err error
		w.parent, err = newWriter(w.path, w.frameThresh, w.level+1)
		if err != nil {
			return err
		}
	}
	return w.parent.writeIndexRecord(key, offset)
}

func (w *Writer) writeIndexRecord(key *zng.Value, offset int64) error {
	if w.builder == nil {
		cols := []zng.Column{
			{"key", key.Type},
			{"value", zng.TypeInt64},
		}
		w.recType = resolver.NewContext().LookupTypeRecord(cols)
		w.builder = zng.NewBuilder(w.recType)
	}
	if w.recType.Columns[0].Type != key.Type {
		return errors.New("zdx error: type of key changed")
	}
	val := zng.EncodeInt(offset)
	return w.Write(w.builder.Build(key.Bytes, val))
}
