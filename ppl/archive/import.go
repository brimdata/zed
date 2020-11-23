package archive

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
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
	ark      *Archive
	cancel   context.CancelFunc
	ctx      context.Context
	errgroup *errgroup.Group
	mu       sync.RWMutex
	once     sync.Once
	writers  map[tsDir]*tsDirWriter

	memBuffered int64
	stats       ImportStats
}

func NewWriter(ctx context.Context, ark *Archive) *Writer {
	g, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	return &Writer{
		ark:      ark,
		cancel:   cancel,
		ctx:      ctx,
		errgroup: g,
		writers:  make(map[tsDir]*tsDirWriter),
	}
}

func (w *Writer) Write(rec *zng.Record) error {
	w.once.Do(func() {
		w.errgroup.Go(w.flusher)
	})
	if w.ctx.Err() != nil {
		return w.errgroup.Wait()
	}

	tsDir := newTsDir(rec.Ts())
	w.mu.RLock()
	dw, ok := w.writers[tsDir]
	w.mu.RUnlock()
	if !ok {
		var err error
		dw, err = newTsDirWriter(w, tsDir)
		if err != nil {
			return err
		}
		w.mu.Lock()
		w.writers[tsDir] = dw
		w.mu.Unlock()
	}
	if err := dw.Write(rec); err != nil {
		return err
	}
	for w.memBuffered > ImportBufSize {
		if err := w.spillLargestBuffer(); err != nil {
			return err
		}
	}
	return nil
}

// flusher is a background goroutine for an active Writer that periodically
// checks for tsdir writers that have not received data in a while. If such a
// writer is found, it is flushed to disk and closed.
func (w *Writer) flusher() error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := w.flushStaleWriters(); err != nil {
				return err
			}
		case <-w.ctx.Done():
			return nil
		}
	}
}

func (w *Writer) flushStaleWriters() error {
	w.mu.Lock()
	var stale []*tsDirWriter
	now := time.Now()
	for tsd, writer := range w.writers {
		if now.Sub(writer.modified()) > w.ark.ImportFlushTimeout {
			stale = append(stale, writer)
			delete(w.writers, tsd)
		}
	}
	w.mu.Unlock()
	for _, writer := range stale {
		if err := writer.flush(); err != nil {
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
	w.mu.RLock()
	defer w.mu.RUnlock()
	var largest *tsDirWriter
	for _, dw := range w.writers {
		if largest == nil || dw.bufSize > largest.bufSize {
			largest = dw
		}
	}
	return largest.spill()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	var merr error
	for ts, dw := range w.writers {
		if err := dw.flush(); err != nil {
			merr = multierr.Append(merr, err)
		}
		delete(w.writers, ts)
	}
	w.mu.Unlock()
	w.cancel()
	if err := w.errgroup.Wait(); merr == nil {
		merr = err
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
	modts   int64
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

func (dw *tsDirWriter) Write(rec *zng.Record) error {
	// XXX This call leads to a ton of one-off allocations that burden the GC
	// and slow down import. We should instead copy the raw record bytes into a
	// recycled buffer and keep around an array of ts + byte-slice structs for
	// sorting.
	rec.CopyBody()
	dw.records = append(dw.records, rec)
	dw.addBufSize(int64(len(rec.Raw)))
	dw.touch()
	if dw.chunkSizeEstimate() > dw.ark.LogSizeThreshold {
		if err := dw.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (dw *tsDirWriter) touch() {
	atomic.StoreInt64(&dw.modts, int64(nano.Now()))
}

func (dw *tsDirWriter) modified() time.Time {
	return nano.Ts(atomic.LoadInt64(&dw.modts)).Time()
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
