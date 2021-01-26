package chunk

import (
	"context"
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/archive/index"
	"github.com/brimsec/zq/ppl/archive/seekindex"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/segmentio/ksuid"
)

// Writer is a zbuf.Writer that writes a stream of sorted records into an
// archive chunk file.
type Writer struct {
	byteCounter    *writeCounter
	count          uint64
	chunk          Chunk
	dataFileWriter *zngio.Writer
	firstTs        nano.Ts
	id             ksuid.KSUID
	indexWriter    indexWriter
	lastTs         nano.Ts
	masks          []ksuid.KSUID
	needSeekWrite  bool
	order          zbuf.Order
	seekIndex      *seekindex.Builder
	dir            iosrc.URI
	wroteFirst     bool
}

type WriterOpts struct {
	Definitions index.Definitions
	Masks       []ksuid.KSUID
	Order       zbuf.Order
	Zng         zngio.WriterOpts
}

func NewWriter(ctx context.Context, dir iosrc.URI, opts WriterOpts) (*Writer, error) {
	id := ksuid.New()
	out, err := iosrc.NewWriter(ctx, dir.AppendPath(ChunkFileName(id)))
	if err != nil {
		return nil, err
	}
	counter := &writeCounter{bufwriter.New(out), 0}
	dataFileWriter := zngio.NewWriter(counter, opts.Zng)
	seekIndexPath := chunkSeekIndexPath(dir, id)
	seekIndex, err := seekindex.NewBuilder(ctx, seekIndexPath.String(), opts.Order)
	if err != nil {
		return nil, err
	}
	idxWriter := indexWriter(nopIndexWriter{})
	// For a Chunk writer we only care about writing index defs whose input
	// path is empty (i.e. references the chunk data itself).
	if defs := opts.Definitions.StandardInputs(); len(defs) > 0 {
		dir := chunkZarDir(dir, id)
		idxWriter, err = index.NewMultiWriter(ctx, dir, defs)
		if err != nil {
			return nil, err
		}
	}
	return &Writer{
		byteCounter:    counter,
		dataFileWriter: dataFileWriter,
		id:             id,
		indexWriter:    idxWriter,
		seekIndex:      seekIndex,
		masks:          opts.Masks,
		order:          opts.Order,
		dir:            dir,
	}, nil
}

type indexWriter interface {
	zbuf.WriteCloser
	Abort()
}

type nopIndexWriter struct{}

func (n nopIndexWriter) Write(*zng.Record) error { return nil }
func (n nopIndexWriter) Close() error            { return nil }
func (n nopIndexWriter) Abort()                  {}

func (cw *Writer) Position() (int64, nano.Ts, nano.Ts) {
	return cw.dataFileWriter.Position(), cw.firstTs, cw.lastTs
}

func (cw *Writer) Write(rec *zng.Record) error {
	// We want to index the start of stream (SOS) position of the data file by
	// record timestamps; we don't know when we've started a new stream until
	// after we written the first record in the stream.
	sos := cw.dataFileWriter.LastSOS()
	if err := cw.dataFileWriter.Write(rec); err != nil {
		return err
	}
	ts := rec.Ts()
	if !cw.wroteFirst || (cw.needSeekWrite && ts != cw.lastTs) {
		if err := cw.seekIndex.Enter(ts, sos); err != nil {
			return err
		}
		cw.needSeekWrite = false
	}
	if cw.dataFileWriter.LastSOS() != sos {
		cw.needSeekWrite = true
	}
	if !cw.wroteFirst {
		cw.firstTs = ts
		cw.wroteFirst = true
	}
	if err := cw.indexWriter.Write(rec); err != nil {
		return err
	}
	cw.lastTs = ts
	cw.count++
	return nil
}

// abort should be called when an error occurs during write. Errors are ignored
// because the write error will be more informative and should be returned.
func (cw *Writer) Abort() {
	cw.dataFileWriter.Close()
	cw.seekIndex.Abort()
	cw.indexWriter.Abort()
}

func (cw *Writer) Close(ctx context.Context) error {
	return cw.CloseWithTs(ctx, cw.firstTs, cw.lastTs)
}

func (cw *Writer) CloseWithTs(ctx context.Context, firstTs, lastTs nano.Ts) error {
	err := cw.dataFileWriter.Close()
	if err != nil {
		cw.Abort()
		return err
	}
	metadata := Metadata{
		First:       firstTs,
		Last:        lastTs,
		RecordCount: cw.count,
		Masks:       cw.masks,
		Size:        cw.dataFileWriter.Position(),
	}
	mdPath := MetadataPath(cw.dir, cw.id)
	if err := metadata.Write(ctx, mdPath, cw.order); err != nil {
		cw.Abort()
		return fmt.Errorf("failed to write chunk metadata to %v: %w", mdPath, err)
	}
	if err := cw.seekIndex.Close(); err != nil {
		cw.Abort()
		return err
	}
	if err := cw.indexWriter.Close(); err != nil {
		return err
	}
	// TODO: zq#1264
	// Add an entry to the update log for S3 backed stores containing the
	// location of the just added data & index file.
	cw.chunk = metadata.Chunk(cw.dir, cw.id)
	return nil
}

func (cw *Writer) BytesWritten() int64 {
	return cw.byteCounter.size
}

func (cw *Writer) RecordsWritten() uint64 {
	return cw.count
}

// Chunk returns the Chunk written by the writer. This is only valid after
// Close() has returned a nil error.
func (cw *Writer) Chunk() Chunk {
	return cw.chunk
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
