package index

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

// Writer implements the zio.Writer interface. A Writer creates a Zed index,
// comprising the base zng file along with its related B-tree sections,
// as zng records are consumed.
//
// The keyFields argument to NewWriter provides a list of the key names, ordered by
// precedence, that will serve as the keys into the index.  The input
// records may or may not have all the key fields.  If a key field is
// missing, it appears as a null value in the index.  Nulls are sorted
// before all non-null values.  All key fields must have the same type.
// The Writer may detect an error if a key field changes type but does not
// check that every key has the same type; it is up to the caller to guarantee
// this type consistency.  For example, the caller should create a separate
// index for fields that have a common name but different types.
//
// The keys in the input zng stream must be previously sorted consistent
// with the precedence order of the keyFields.
//
// As the zng file data is written, a B-tree index is computed as a
// constant B-tree to make key lookups efficient.  The B-tree sections
// are written to temporary files and at close, they are merged into
// a single-file index.
//
// If a Writer is created but Closed without ever writing records to it, then
// the index is created with no keys and an "empty" index trailer.  This is
// useful for knowing when something has been indexed but no keys were present.
// If a Writer is created then an error is enountered (for example, the type of
// key changes), then you generally want to abort and cleanup by calling Abort()
// instead of Close().
type Writer struct {
	uri         *storage.URI
	keyer       *Keyer
	zctx        *zed.Context
	ectx        expr.Context
	engine      storage.Engine
	writer      *indexWriter
	tmpdir      string
	frameThresh int
	iow         io.WriteCloser
	childField  string
	nlevel      int
	order       order.Which
}

type indexWriter struct {
	base       *Writer
	parent     *indexWriter
	ectx       expr.Context
	name       string
	buffer     *bufwriter.Writer
	zng        *zngio.Writer
	frameStart int64
	frameEnd   int64
	frameKey   *zed.Value
}

// NewWriter returns a Writer ready to write a Zed index or it returns
// an error.  The index is written to the URL provided in the path argument
// while temporary file are written locally.  Calls to Write must
// provide keys in increasing lexicographic order.  Duplicate keys are not
// allowed but will not be detected.  Close() or Abort() must be called when
// done writing.
func NewWriter(zctx *zed.Context, engine storage.Engine, path string, keys field.List, options ...Option) (*Writer, error) {
	return NewWriterWithContext(context.Background(), zctx, engine, path, keys, options...)
}

func NewWriterWithContext(ctx context.Context, zctx *zed.Context, engine storage.Engine, path string, keys field.List, options ...Option) (*Writer, error) {
	if len(keys) == 0 {
		return nil, errors.New("must specify at least one key")
	}
	w := &Writer{
		zctx:       zctx,
		ectx:       expr.NewContext(),
		engine:     engine,
		childField: uniqChildField(keys),
	}
	for _, opt := range options {
		opt.apply(w)
	}
	if w.frameThresh == 0 {
		w.frameThresh = frameThresh
	}
	if w.frameThresh > FrameMaxSize {
		return nil, fmt.Errorf("frame threshold too large (%d)", w.frameThresh)
	}
	var err error
	w.uri, err = storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	w.iow, err = engine.Put(ctx, w.uri)
	if err != nil {
		return nil, err
	}
	w.tmpdir, err = os.MkdirTemp("", "zed-index-*")
	if err != nil {
		return nil, err
	}
	w.keyer, err = NewKeyer(zctx, keys)
	return w, err
}

type Option interface {
	apply(*Writer)
}

type optionFunc func(*Writer)

func (f optionFunc) apply(w *Writer) { f(w) }

func FrameThresh(frameThresh int) Option {
	return optionFunc(func(w *Writer) {
		w.frameThresh = frameThresh
	})
}

func Order(o order.Which) Option {
	return optionFunc(func(w *Writer) {
		w.order = o
	})
}

func (w *Writer) Write(val *zed.Value) error {
	if w.writer == nil {
		var err error
		w.writer, err = newIndexWriter(w, w.iow, "", w.ectx)
		if err != nil {
			return err
		}
		// Check that key is present... ?!
		if _, err := w.keyer.KeyOf(w.ectx, val); err != nil {
			return err
		}
	}
	return w.writer.write(val)
}

// Abort closes this writer, deleting any and all objects and/or files associated
// with it.
func (w *Writer) Abort() error {
	// Delete the temp files comprising the index hierarchy.
	defer os.RemoveAll(w.tmpdir)
	err := w.closeTree()
	if closeErr := w.iow.Close(); err == nil {
		err = closeErr
	}
	// Ignore context here in the event that context is the reson for the abort.
	if rmErr := w.engine.Delete(context.Background(), w.uri); err == nil {
		err = rmErr
	}
	return err
}

func (w *Writer) Close() error {
	// No matter what, delete the temp files comprising the index hierarchy.
	defer os.RemoveAll(w.tmpdir)
	// First, close the parent if it exists (which will recursively close
	// all the parents to the root) while leaving the base layer open.
	if err := w.closeTree(); err != nil {
		w.iow.Close()
		return err
	}
	if w.writer == nil {
		// If the writer hasn't been created because no records were
		// encountered, then the base layer writer was never created.
		// In this case, bypass the base layer, write an empty trailer
		// directly to the output, and close.
		zw := zngio.NewWriter(w.iow, zngio.WriterOpts{})
		err := writeTrailer(zw, w.zctx, w.childField, w.frameThresh, nil, w.keyer.Keys(), w.order)
		if err2 := w.iow.Close(); err == nil {
			err = err2
		}
		return err
	}
	// Otherwise, close the frame of the base layer so we can copy the hierarchy
	// to the base.  Note that sum of the sizes of the parents is much smaller
	// than the base so this will go fast compared to the entire indexing job.
	if err := w.writer.closeFrame(); err != nil {
		return err
	}
	// The hierarchy is now flushed and closed.  Assemble the file into
	// a single index and remove the temporary btree files.
	if err := w.finalize(); err != nil {
		w.writer.buffer.Close()
		return err
	}
	// Finally, close the base layer.
	return w.writer.buffer.Close()
}

func (w *Writer) closeTree() error {
	if w.writer == nil {
		return nil
	}
	var err error
	for p := w.writer.parent; p != nil; p = p.parent {
		if closeErr := p.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func (w *Writer) finalize() error {
	// First, collect up parent linkage into a slice so we can traverse
	// top down...
	base := w.writer
	var layers []*indexWriter
	for p := base.parent; p != nil; p = p.parent {
		layers = append(layers, p)
	}
	// Now, copy each non-base file in top-down order to the base-layer object.
	var sizes []int64
	sizes = append(sizes, base.frameStart)
	for k := len(layers) - 1; k >= 0; k-- {
		// Copy the files in the reverse order so the root comes first.
		// This will avoid backward seeks while looking up keys in the tree
		// (except for the one backward seek to the base layer).
		layer := layers[k]
		size := layer.frameStart
		sizes = append(sizes, size)
		f, err := os.Open(layer.name)
		if err != nil {
			return err
		}
		n, err := io.Copy(base.buffer, f)
		if err != nil {
			f.Close()
			return err
		}
		if n != size {
			f.Close()
			return fmt.Errorf("internal Zed index error: index file size (%d) does not equal zng size (%d)", n, size)
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return writeTrailer(base.zng, w.zctx, w.childField, w.frameThresh, sizes, w.keyer.Keys(), w.order)
}

func writeTrailer(w *zngio.Writer, zctx *zed.Context, childField string, frameThresh int, sizes []int64, keys field.List, o order.Which) error {
	t := Trailer{
		ChildOffsetField: childField,
		FrameThresh:      frameThresh,
		Keys:             keys,
		Magic:            Magic,
		Order:            o,
		Sections:         sizes,
		Version:          Version,
	}
	rec, err := zson.NewZNGMarshalerWithContext(zctx).MarshalRecord(t)
	if err != nil {
		return err
	}
	if err := w.Write(rec); err != nil {
		return err
	}
	return w.EndStream()
}

func newIndexWriter(base *Writer, w io.WriteCloser, name string, ectx expr.Context) (*indexWriter, error) {
	base.nlevel++
	if base.nlevel >= MaxLevels {
		return nil, ErrTooManyLevels
	}
	writer := bufwriter.New(w)
	return &indexWriter{
		base:     base,
		buffer:   writer,
		ectx:     ectx,
		name:     name,
		zng:      zngio.NewWriter(writer, zngio.WriterOpts{}),
		frameEnd: int64(base.frameThresh),
	}, nil
}

func (w *indexWriter) newParent() (*indexWriter, error) {
	file, err := os.CreateTemp(w.base.tmpdir, "")
	if err != nil {
		return nil, err
	}
	return newIndexWriter(w.base, file, file.Name(), w.ectx)
}

func (w *indexWriter) Close() error {
	// Make sure to pass up framekeys to parent trees, even though frames aren't
	// full.
	if err := w.closeFrame(); err != nil {
		return err
	}
	return w.buffer.Close()
}

func (w *indexWriter) write(rec *zed.Value) error {
	offset := w.zng.Position()
	if offset >= w.frameEnd && w.frameKey != nil {
		// the frame in place is already big enough... flush it and
		// start going on the next
		if err := w.endFrame(); err != nil {
			return err
		}
		// endFrame will close the frame which will reset
		// frameStart
		w.frameEnd = w.frameStart + int64(w.base.frameThresh)
	}
	if w.frameKey == nil {
		// When we start a new frame, we want to create a key entry
		// in the parent for the current key but we don't want to write
		// it until we know this frame will be big enough to add it
		// (or until we know it's the last frame in the file).
		// So we build the frame key record from the current record
		// here ahead of its use and save it in the frameKey variable.
		key, err := w.base.keyer.KeyOf(w.ectx, rec)
		// If the key isn't here flag an error.  All keys must be
		// present to build a proper index.
		if err != nil {
			return err
		}
		w.frameKey = key.Copy()
	}
	return w.zng.Write(rec)
}

func (w *indexWriter) endFrame() error {
	if err := w.addToParentIndex(w.frameKey, w.frameStart); err != nil {
		return err
	}
	if err := w.closeFrame(); err != nil {
		return err
	}
	return nil
}

func (w *indexWriter) closeFrame() error {
	if err := w.zng.EndStream(); err != nil {
		return err
	}
	w.frameStart = w.zng.Position()
	w.frameKey = nil
	return nil
}

func (w *indexWriter) addToParentIndex(key *zed.Value, offset int64) error {
	if w.parent == nil {
		var err error
		w.parent, err = w.newParent()
		if err != nil {
			return err
		}
	}
	return w.parent.writeIndexRecord(key, offset)
}

func (w *indexWriter) writeIndexRecord(keys *zed.Value, offset int64) error {
	col := []zed.Column{{w.base.childField, zed.TypeInt64}}
	val := zed.EncodeInt(offset)
	rec, err := w.base.zctx.AddColumns(keys, col, []zed.Value{{zed.TypeInt64, val}})
	if err != nil {
		return err
	}
	return w.write(rec)
}
