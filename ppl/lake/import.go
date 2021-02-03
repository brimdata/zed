package lake

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/lake/chunk"
	"github.com/brimsec/zq/ppl/lake/index"
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

	// For unit testing.
	importLZ4BlockSize = zngio.DefaultLZ4BlockSize
)

const importDefaultStaleDuration = time.Second * 5

func Import(ctx context.Context, lk *Lake, zctx *resolver.Context, r zbuf.Reader) error {
	w, err := NewWriter(ctx, lk)
	if err != nil {
		return err
	}
	err = zbuf.CopyWithContext(ctx, w, r)
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
type Writer struct {
	lk            *Lake
	cancel        context.CancelFunc
	ctx           context.Context
	defs          index.Definitions
	errgroup      *errgroup.Group
	mu            sync.Mutex
	once          sync.Once
	staleDuration time.Duration
	writers       map[tsDir]*tsDirWriter

	memBuffered int64
	stats       ImportStats
}

// NewWriter creates a zbuf.Writer compliant writer for writing data to an
// archive.
func NewWriter(ctx context.Context, lk *Lake) (*Writer, error) {
	defs, err := lk.ReadDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)
	return &Writer{
		lk:            lk,
		cancel:        cancel,
		ctx:           ctx,
		errgroup:      g,
		defs:          defs,
		staleDuration: importDefaultStaleDuration,
		writers:       make(map[tsDir]*tsDirWriter),
	}, nil
}

// SetStaleDuration sets the stale threshold for the writer which is the delay
// after the last write to a tsdir until it is flushed and removed.
func (w *Writer) SetStaleDuration(dur time.Duration) {
	w.staleDuration = dur
}

func (w *Writer) Write(rec *zng.Record) error {
	w.once.Do(func() {
		w.errgroup.Go(w.flusher)
	})
	if w.ctx.Err() != nil {
		if err := w.errgroup.Wait(); err != nil {
			return err
		}
		return w.ctx.Err()
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	tsDir := newTsDir(rec.Ts())
	dw, ok := w.writers[tsDir]
	if !ok {
		var err error
		dw, err = newTsDirWriter(w, tsDir)
		if err != nil {
			return err
		}
		if _, ok := w.writers[tsDir]; !ok {
			w.writers[tsDir] = dw
		}
		dw = w.writers[tsDir]
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
// checks and closes stale tsdirs.
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

// flushStateWriters loops through the open tsdir writers and flushes and
// removes any writers that haven't recieved any updates since
// Writer.staleDuration.
func (w *Writer) flushStaleWriters() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	var stale []*tsDirWriter
	for tsd, writer := range w.writers {
		if now.Sub(writer.modts.Time()) >= w.staleDuration {
			stale = append(stale, writer)
			delete(w.writers, tsd)
		}
	}

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
// lk.LogSizeThreshold, they are written to a chunk file in
// the archive.
type tsDirWriter struct {
	lk      *Lake
	bufSize int64
	ctx     context.Context
	modts   nano.Ts
	records []*zng.Record
	spiller *spill.MergeSort
	tsDir   tsDir
	writer  *Writer
}

func newTsDirWriter(w *Writer, tsDir tsDir) (*tsDirWriter, error) {
	d := &tsDirWriter{
		lk:     w.lk,
		ctx:    w.ctx,
		tsDir:  tsDir,
		writer: w,
	}
	if err := iosrc.MkdirAll(tsDir.path(w.lk), 0755); err != nil {
		return nil, err
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
	if dw.chunkSizeEstimate() > dw.lk.LogSizeThreshold {
		if err := dw.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (dw *tsDirWriter) touch() {
	dw.modts = nano.Now()
}

func (dw *tsDirWriter) spill() error {
	if len(dw.records) == 0 {
		return nil
	}
	if dw.spiller == nil {
		var err error
		dw.spiller, err = spill.NewMergeSort(importCompareFn(dw.lk))
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
		expr.SortStable(dw.records, importCompareFn(dw.lk))
		r = zbuf.Array(dw.records).NewReader()
	}
	w, err := chunk.NewWriter(dw.ctx, dw.tsDir.path(dw.lk), chunk.WriterOpts{
		Order:       dw.lk.DataOrder,
		Definitions: dw.writer.defs,
		Zng: zngio.WriterOpts{
			StreamRecordsMax: ImportStreamRecordsMax,
			LZ4BlockSize:     importLZ4BlockSize,
		},
	})
	if err != nil {
		return err
	}
	if err := zbuf.CopyWithContext(dw.ctx, w, r); err != nil {
		w.Abort()
		return err
	}
	if err := w.Close(dw.ctx); err != nil {
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

func importCompareFn(lk *Lake) expr.CompareFn {
	return zbuf.NewCompareFn(field.New("ts"), lk.DataOrder == zbuf.OrderDesc)
}
