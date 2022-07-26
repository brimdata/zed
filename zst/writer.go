package zst

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/zst/column"
)

const (
	MaxSegmentThresh = column.MaxSegmentThresh
	MaxSkewThresh    = 512 * 1024 * 1024
)

// Writer implements the zio.Writer interface. A Writer creates a columnar
// zst object from a stream of zed.Records.
type Writer struct {
	zctx       *zed.Context
	writer     io.WriteCloser
	spiller    *column.Spiller
	typeMap    map[zed.Type]int
	writers    []column.Writer
	types      []zed.Type
	skewThresh int
	segThresh  int
	// We keep track of the size of rows we've encoded into in-memory
	// data structures.  This is roughtly propertional to the amount of
	// memory used and the max amount of skew between rows that will be
	// needed for reader-side buffering.  So when the memory footprint
	// exceeds the confired skew theshhold, we flush the columns to storage.
	footprint int
	root      *column.Int64Writer
}

func NewWriter(w io.WriteCloser, skewThresh, segThresh int) (*Writer, error) {
	if err := checkThresh("skew", MaxSkewThresh, skewThresh); err != nil {
		return nil, err
	}
	if err := checkThresh("column", MaxSegmentThresh, segThresh); err != nil {
		return nil, err
	}
	spiller := column.NewSpiller(w, segThresh)
	return &Writer{
		zctx:       zed.NewContext(),
		spiller:    spiller,
		writer:     w,
		typeMap:    make(map[zed.Type]int),
		skewThresh: skewThresh,
		segThresh:  segThresh,
		root:       column.NewInt64Writer(spiller),
	}, nil
}

func (w *Writer) Close() error {
	firstErr := w.finalize()
	if err := w.writer.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

type WriterURI struct {
	Writer
	engine storage.Engine
	uri    *storage.URI
}

func NewWriterFromPath(ctx context.Context, engine storage.Engine, path string, skewThresh, segThresh int) (*WriterURI, error) {
	uri, err := storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	w, err := engine.Put(ctx, uri)
	if err != nil {
		return nil, err
	}
	writer, err := NewWriter(bufwriter.New(w), skewThresh, segThresh)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", uri, err)
	}
	return &WriterURI{
		Writer: *writer,
		engine: engine,
		uri:    uri,
	}, nil
}

func checkThresh(which string, max, thresh int) error {
	if thresh == 0 {
		return fmt.Errorf("ZST %s threshold cannot be zero", which)
	}
	if thresh > max {
		return fmt.Errorf("ZST %s threshold too large (%d)", which, thresh)
	}
	return nil
}

func (w *Writer) Write(val *zed.Value) error {
	// The ZST writer self-organizes around the types that are
	// written to it.  No need to define the schema up front!
	// We track the types seen first-come, first-served in the
	// typeMap table and the ZST serialization structure
	// follows accordingly.
	typ := val.Type
	typeNo, ok := w.typeMap[typ]
	if !ok {
		typeNo = len(w.types)
		w.typeMap[typ] = typeNo
		w.writers = append(w.writers, column.NewWriter(typ, w.spiller))
		w.types = append(w.types, val.Type)
	}
	if err := w.root.Write(int64(typeNo)); err != nil {
		return err
	}
	if err := w.writers[typeNo].Write(val.Bytes); err != nil {
		return err
	}
	w.footprint += len(val.Bytes)
	if w.footprint >= w.skewThresh {
		w.footprint = 0
		return w.flush(false)
	}
	return nil
}

// Abort closes this writer, deleting any and all objects and/or files associated
// with it.
func (w *WriterURI) Abort(ctx context.Context) error {
	firstErr := w.writer.Close()
	if err := w.engine.DeleteByPrefix(ctx, w.uri); firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (w *WriterURI) Close() error {
	if err := w.finalize(); err != nil {
		w.writer.Close()
		return err
	}
	return w.writer.Close()
}

func (w *Writer) flush(eof bool) error {
	for _, writer := range w.writers {
		if err := writer.Flush(eof); err != nil {
			return err
		}
	}
	return w.root.Flush(eof)
}

func (w *Writer) finalize() error {
	if err := w.flush(true); err != nil {
		return err
	}
	// At this point all the column data has been written out
	// to the underlying spiller, so we start writing zng at this point.
	zw := zngio.NewWriter(w.writer)
	dataSize := w.spiller.Position()
	// First, we write the root segmap of the vector of integer type IDs.
	m := zson.NewZNGMarshalerWithContext(w.zctx)
	m.Decorate(zson.StyleSimple)
	val, err := m.Marshal(w.root.Segmap())
	if err != nil {
		//XXX wrap
		return err
	}
	if err := zw.Write(val); err != nil {
		//XXX wrap
		return err
	}
	// Now, write the reassembly maps for each top-level type.
	for _, writer := range w.writers {
		val, err = m.Marshal(writer.Metadata())
		if err != nil {
			//XXX wrap
			return err
		}
		if err := zw.Write(val); err != nil {
			//XXX wrap
			return err
		}
	}
	zw.EndStream()
	mapsSize := zw.Position()
	// Finally, we build and write the section trailer based on the size
	// of the data and size of the reassembly maps.
	sections := []int64{dataSize, mapsSize}
	trailer, err := zngio.MarshalTrailer(FileType, Version, sections, &FileMeta{w.skewThresh, w.segThresh})
	if err != nil {
		return err
	}
	if err := zw.Write(&trailer); err != nil {
		return err
	}
	return zw.EndStream()
}
