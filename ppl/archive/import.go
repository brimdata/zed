package archive

import (
	"context"
	"sync/atomic"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/multierr"
)

var (
	// ImportBufSize specifies the max size of the records buffered during import
	// before they are flushed to disk.
	ImportBufSize          = int64(sort.MemMaxBytes)
	ImportStreamRecordsMax = zngio.DefaultStreamRecordsMax
)

// For unit testing.
var importLZ4BlockSize = zngio.DefaultLZ4BlockSize

func Import(ctx context.Context, ark *Archive, zctx *resolver.Context, r zbuf.Reader) error {
	w := NewWriter(ctx, ark)
	err := zbuf.CopyWithContext(ctx, w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Writer is a zbuf.Writer that partitions records by day into the
// appropriate tsDirWriter. Writer keeps track of the overall memory
// footprint of the collection of tsDirWriter and instructs the tsDirWriter
// with the largest footprint to spill its records to a temporary file on disk.
//
// TODO zq#1432 Writer does not currently keep track of size of records
// written to temporary files. At some point this should have a maxTempFileSize
// to ensure the Writer does not exceed the size of a provisioned tmpfs.
//
// TODO zq#1433 If a tsDir never gets enough data to reach ark.LogSizeThreshold,
// the data will sit in the tsDirWriter and remain unsearchable until the
// provided read stream is closed. Add some kind of timeout functionality that
// periodically flushes stale tsDirWriters.
type Writer struct {
	ark     *Archive
	ctx     context.Context
	writers map[tsDir]*tsDirWriter

	memBuffered int64
	stats       ImportStats
}

func NewWriter(ctx context.Context, ark *Archive) *Writer {
	return &Writer{
		ark:     ark,
		ctx:     ctx,
		writers: make(map[tsDir]*tsDirWriter),
	}
}

func (w *Writer) Write(rec *zng.Record) error {
	tsDir := newTsDir(rec.Ts())
	dw, ok := w.writers[tsDir]
	if !ok {
		var err error
		dw, err = newTsDirWriter(w, tsDir)
		if err != nil {
			return err
		}
		w.writers[tsDir] = dw
	}
	if err := dw.writeOne(rec); err != nil {
		return err
	}
	for w.memBuffered > ImportBufSize {
		if err := w.spillLargestBuffer(); err != nil {
			return err
		}
	}
	return nil
}

// spillLargestBuffer is called when the total size of buffered records exceeeds
// ImportBufSize. spillLargestBuffer attempts to clear up memory in use by
// spilling to disk the records of the tsDirWriter with the largest memory
// footprint.
func (w *Writer) spillLargestBuffer() error {
	var largest *tsDirWriter
	for _, dw := range w.writers {
		if largest == nil || dw.bufSize > largest.bufSize {
			largest = dw
		}
	}
	return largest.spill()
}

func (w *Writer) Close() error {
	var merr error
	for ts, dw := range w.writers {
		if err := dw.flush(); err != nil {
			merr = multierr.Append(merr, err)
		}
		delete(w.writers, ts)
	}
	return merr
}

func (w *Writer) Stats() ImportStats {
	return w.stats.Copy()
}

type ImportStats struct {
	DataChunksWritten  int64
	RecordBytesWritten int64
	RecordsWritten     int64
}

func (s *ImportStats) Accumulate(b ImportStats) {
	atomic.AddInt64(&s.DataChunksWritten, b.DataChunksWritten)
	atomic.AddInt64(&s.RecordBytesWritten, b.RecordBytesWritten)
	atomic.AddInt64(&s.RecordsWritten, b.RecordsWritten)
}

func (s *ImportStats) Copy() ImportStats {
	return ImportStats{
		DataChunksWritten:  atomic.LoadInt64(&s.DataChunksWritten),
		RecordBytesWritten: atomic.LoadInt64(&s.RecordBytesWritten),
		RecordsWritten:     atomic.LoadInt64(&s.RecordsWritten),
	}
}

// tsDirWriter accumulates records for one tsDir.
// When the expected size of writing the records is greater than
// ark.LogSizeThreshold, they are written to a chunk file in
// the archive.
type tsDirWriter struct {
	ark     *Archive
	bufSize int64
	ctx     context.Context
	records []*zng.Record
	spiller *spill.MergeSort
	tsDir   tsDir
	writer  *Writer
}

func newTsDirWriter(w *Writer, tsDir tsDir) (*tsDirWriter, error) {
	d := &tsDirWriter{
		ark:    w.ark,
		ctx:    w.ctx,
		tsDir:  tsDir,
		writer: w,
	}
	if dirmkr, ok := d.ark.dataSrc.(iosrc.DirMaker); ok {
		if err := dirmkr.MkdirAll(tsDir.path(w.ark), 0755); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (dw *tsDirWriter) addBufSize(delta int64) {
	dw.bufSize += delta
	dw.writer.memBuffered += delta
}

// chunkSizeEstimate returns a crude approximation of all records when written
// to a chunk file (i.e. compressed).
func (dw *tsDirWriter) chunkSizeEstimate() int64 {
	b := dw.bufSize
	if dw.spiller != nil {
		b += dw.spiller.SpillSize()
	}
	return b / 2
}

func (dw *tsDirWriter) writeOne(rec *zng.Record) error {
	// XXX This call leads to a ton of one-off allocations that burden the GC
	// and slow down import. We should instead copy the raw record bytes into a
	// recycled buffer and keep around an array of ts + byte-slice structs for
	// sorting.
	rec.CopyBody()
	dw.records = append(dw.records, rec)
	dw.addBufSize(int64(len(rec.Raw)))
	if dw.chunkSizeEstimate() > dw.ark.LogSizeThreshold {
		if err := dw.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (dw *tsDirWriter) spill() error {
	if len(dw.records) == 0 {
		return nil
	}
	if dw.spiller == nil {
		var err error
		dw.spiller, err = spill.NewMergeSort(importCompareFn(dw.ark))
		if err != nil {
			return err
		}
	}
	if err := dw.spiller.Spill(dw.records); err != nil {
		return err
	}
	dw.records = dw.records[:0]
	dw.addBufSize(dw.bufSize * -1)
	return nil
}

func (dw *tsDirWriter) flush() error {
	var r zbuf.Reader
	if dw.spiller != nil {
		if err := dw.spill(); err != nil {
			return err
		}
		spiller := dw.spiller
		dw.spiller, r = nil, spiller
		defer spiller.Cleanup()
	} else {
		// If len of records is 0 and spiller is nil, the tsDirWriter is empty.
		// Just return nil.
		if len(dw.records) == 0 {
			return nil
		}
		expr.SortStable(dw.records, importCompareFn(dw.ark))
		r = zbuf.Array(dw.records).NewReader()
	}
	w, err := chunk.NewWriter(dw.ctx, dw.tsDir.path(dw.ark), dw.ark.DataOrder, nil, zngio.WriterOpts{
		StreamRecordsMax: ImportStreamRecordsMax,
		LZ4BlockSize:     importLZ4BlockSize,
	})
	if err != nil {
		return err
	}
	if err := zbuf.CopyWithContext(dw.ctx, w, r); err != nil {
		w.Abort()
		return err
	}
	if _, err := w.Close(dw.ctx); err != nil {
		return err
	}
	dw.writer.stats.Accumulate(ImportStats{
		DataChunksWritten:  1,
		RecordBytesWritten: w.BytesWritten(),
		RecordsWritten:     int64(w.RecordsWritten()),
	})
	dw.records = dw.records[:0]
	dw.addBufSize(dw.bufSize * -1)
	return nil
}

func importCompareFn(ark *Archive) expr.CompareFn {
	return zbuf.NewCompareFn(field.New("ts"), ark.DataOrder == zbuf.OrderDesc)
}
