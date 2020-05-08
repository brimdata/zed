package zdx

import (
	"errors"
	"fmt"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Writer implements the zbuf.Writer interface. A Writer creates a zdx index,
// comprising the base zng file along with its related b-tree sections,
// as zng records are consumed.
//
// The keyFields argument to NewWriter provides a list of the key names, ordered by
// precedence, that will serve as the keys into the index.  The input
// records may or may not have all the key fields.  If a key field is
// missing it appears as a null value in the index.  Nulls are sorted
// before all non-null values.
//
// The keys in the input zng stream must be previously sorted consistent
// with the precedence order of the keyFields.
//
// As the zng file data is written, a b-tree index is computed as a
// constant b-tree to make key lookups efficient.  The b-tree sections
// are written to temporary zng files (.1, .2, etc) and at close,
// they are collapsed into the single-file zdx format.
// XXX TBD: the single-file implementation will arrive in a subsequent PR.
type Writer struct {
	zctx        *resolver.Context
	path        string
	level       int
	writer      *bufwriter.Writer
	out         *zngio.Writer
	parent      *Writer
	header      *zng.Record
	frameThresh int
	frameStart  int64
	frameEnd    int64
	frameKey    *zng.Record
	keyFields   []string
	cutter      *proc.Cutter
	recType     *zng.TypeRecord
}

// NewWriter returns a Writer ready to write an zdx bundle and related
// index files via subsequent calls to Write(), or it returns an error.
// All files will be written to the directory indicated by path
// with the form $path, $path.1, and so forth.  Calls to Write must
// provide keys in increasing lexicographic order.  Duplicate keys are not
// allowed but will not be detected.  Close() must be called when done writing.
func NewWriter(zctx *resolver.Context, path string, keyFields []string, framesize int) (*Writer, error) {
	if keyFields == nil {
		keyFields = []string{"key"}
	}
	if framesize == 0 {
		return nil, errors.New("zdx framesize cannot be zero")
	}
	return newWriter(zctx, path, keyFields, framesize, 0, nil)
}

func newWriter(zctx *resolver.Context, path string, keyFields []string, framesize, level int, hdr *zng.Record) (*Writer, error) {
	if level > 5 {
		panic("something wrong")
	}
	name := filename(path, level)
	f, err := fs.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	writer := bufwriter.New(f)
	return &Writer{
		zctx:        zctx,
		path:        path,
		keyFields:   keyFields,
		level:       level,
		writer:      writer,
		out:         zngio.NewWriter(writer, zio.WriterFlags{}),
		header:      hdr,
		frameThresh: framesize,
		frameEnd:    int64(framesize),
		cutter:      proc.NewCutter(zctx, false, keyFields),
	}, nil
}

// Flush implements zbuf.WriteFlusher.
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
	if offset >= w.frameEnd && w.frameKey != nil {
		w.frameEnd = offset + int64(w.frameThresh)
		// the frame in place is already big enough... flush it and
		// start going on the next
		if err := w.endFrame(); err != nil {
			return err
		}
	}
	if w.frameKey == nil {
		// When we start a new frame, we want to create a key entry
		// in the parent for the current key but we don't want to write
		// it until we know this frame will be big enough to add it
		// (or until we know it's the last frame in the file).
		// So we build the frame key record from the current record
		// here ahead of its use and save it in the frameKey variable.
		key, err := w.cutter.Cut(rec)
		if err != nil {
			return err
		}
		// If the key isn't here flag an error.  All keys must be
		// present to build a proper index.
		// XXX We also need to check that they are in order.
		if key == nil {
			return fmt.Errorf("no key field present in record of type: %s", rec.Type)
		}
		w.frameKey = key
		// If this is the start of the level zero zdx, emit the superblock.
		if w.header == nil {
			hdr, err := newHeader(w.zctx, key)
			if err != nil {
				return err
			}
			w.header = hdr
			if err := w.out.Write(hdr); err != nil {
				return err
			}
		}
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

func (w *Writer) addToParentIndex(key *zng.Record, offset int64) error {
	if w.parent == nil {
		var err error
		w.parent, err = newWriter(w.zctx, w.path, w.keyFields, w.frameThresh, w.level+1, w.header)
		if err != nil {
			return err
		}
	}
	return w.parent.writeIndexRecord(key, offset)
}

func (w *Writer) writeIndexRecord(keys *zng.Record, offset int64) error {
	childField, err := w.header.AccessString("child_field")
	if err != nil {
		return err
	}
	col := []zng.Column{{childField, zng.TypeInt64}}
	val := zng.EncodeInt(offset)
	rec, err := w.zctx.AddColumns(keys, col, []zng.Value{{zng.TypeInt64, val}})
	if err != nil {
		return err
	}
	return w.Write(rec)
}
