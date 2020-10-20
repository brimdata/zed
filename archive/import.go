package archive

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
)

// ImportBufSize specifies the max size of the records buffered during import
// before they are flushed to disk.
var ImportBufSize = int64(sort.MemMaxBytes)

// The below are vars for unit testing.
var (
	importLZ4BlockSize     = zngio.DefaultLZ4BlockSize
	importStreamRecordsMax = zngio.DefaultStreamRecordsMax
)

func Import(ctx context.Context, ark *Archive, zctx *resolver.Context, r zbuf.Reader) error {
	w := newImportWriter(ctx, ark)
	err := zbuf.CopyWithContext(ctx, w, r)
	if closeErr := w.close(); err == nil {
		err = closeErr
	}
	return err
}

// importWriter is a zbuf.Writer that partitions records by day into the
// appropriate tsDirWriter. importWriter keeps track of the overall memory
// footprint of the collection of tsDirWriter and instructs the tsDirWriter
// with the largest footprint to spill its records to a temporary file on disk.
//
// TODO zq#1432 importWriter does not currently keep track of size of records
// written to temporary files. At some point this should have a maxTempFileSize
// to ensure the importWriter does not exceed the size of a provisioned tmpfs.
//
// TODO zq#1433 If a tsDir never gets enough data to reach ark.LogSizeThreshold,
// the data will sit in the tsDirWriter and remain unsearchable until the
// provided read stream is closed. Add some kind of timeout functionality that
// periodically flushes stale tsDirWriters.
type importWriter struct {
	ark     *Archive
	ctx     context.Context
	writers map[tsDir]*tsDirWriter

	memBuffered int64
}

func newImportWriter(ctx context.Context, ark *Archive) *importWriter {
	return &importWriter{
		ark:     ark,
		ctx:     ctx,
		writers: make(map[tsDir]*tsDirWriter),
	}
}

func (iw *importWriter) Write(rec *zng.Record) error {
	tsDir := newTsDir(rec.Ts())
	dw, ok := iw.writers[tsDir]
	if !ok {
		var err error
		dw, err = newTsDirWriter(iw, tsDir)
		if err != nil {
			return err
		}
		iw.writers[tsDir] = dw
	}
	if err := dw.writeOne(rec); err != nil {
		return err
	}
	for iw.memBuffered > ImportBufSize {
		if err := iw.spillLargestBuffer(); err != nil {
			return err
		}
	}
	return nil
}

// spillLargestBuffer is called when the total size of buffered records exceeeds
// ImportBufSize. spillLargestBuffer attempts to clear up memory in use by
// spilling to disk the records of the tsDirWriter with the largest memory
// footprint.
func (iw *importWriter) spillLargestBuffer() error {
	var dw *tsDirWriter
	for _, w := range iw.writers {
		if dw == nil || w.bufSize > dw.bufSize {
			dw = w
		}
	}
	return dw.spill()
}

func (iw *importWriter) close() error {
	var merr error
	for ts, w := range iw.writers {
		if err := w.flush(); err != nil {
			merr = multierr.Append(merr, err)
		}
		delete(iw.writers, ts)
	}
	return merr
}

// tsDirWriter accumulates records for one tsDir.
// When the expected size of writing the records is greater than
// ark.LogSizeThreshold, they are written to a chunk file in
// the archive.
type tsDirWriter struct {
	ark          *Archive
	bufSize      int64
	count        uint64
	ctx          context.Context
	importWriter *importWriter
	maxTs        nano.Ts
	minTs        nano.Ts
	records      []*zng.Record
	spiller      *spill.MergeSort
	tsDir        tsDir
}

func newTsDirWriter(iw *importWriter, tsDir tsDir) (*tsDirWriter, error) {
	d := &tsDirWriter{
		ark:          iw.ark,
		ctx:          iw.ctx,
		importWriter: iw,
		tsDir:        tsDir,
		minTs:        nano.MaxTs,
		maxTs:        nano.MinTs,
	}
	if dirmkr, ok := d.ark.dataSrc.(iosrc.DirMaker); ok {
		if err := dirmkr.MkdirAll(tsDir.path(iw.ark), 0755); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (dw *tsDirWriter) addBufSize(delta int64) {
	dw.bufSize += delta
	dw.importWriter.memBuffered += delta
}

// totalRecordBytes is the sum of the size of compressed records spilt to disk
// and a crude approximation of the buffer record bytes (simply bufBytes / 2).
func (dw *tsDirWriter) totalRecordBytes() int64 {
	b := dw.bufSize
	if dw.spiller != nil {
		b += dw.spiller.SpillSize()
	}
	return b
}

func (dw *tsDirWriter) writeOne(rec *zng.Record) error {
	dw.records = append(dw.records, rec)
	ts := rec.Ts()
	if ts < dw.minTs {
		dw.minTs = ts
	}
	if ts > dw.maxTs {
		dw.maxTs = ts
	}
	dw.count++
	dw.addBufSize(int64(len(rec.Raw)))
	if dw.totalRecordBytes() > dw.ark.LogSizeThreshold {
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
	first, last := dw.minTs, dw.maxTs
	if dw.ark.DataOrder == zbuf.OrderDesc {
		first, last = last, first
	}
	chunk := Chunk{
		Id:          ksuid.New(),
		First:       first,
		Last:        last,
		Kind:        FileKindData,
		RecordCount: dw.count,
	}
	w, err := newChunkWriter(dw.ctx, dw.ark, dw.tsDir, chunk)
	if err != nil {
		return err
	}
	if err := zbuf.CopyWithContext(dw.ctx, w, r); err != nil {
		w.abort()
		return err
	}
	if err := w.close(dw.ctx); err != nil {
		return err
	}
	dw.records = dw.records[:0]
	dw.addBufSize(dw.bufSize * -1)
	dw.minTs = nano.MaxTs
	dw.maxTs = nano.MinTs
	dw.count = 0
	return nil
}

// chunkWriter is a zbuf.Writer that writes a stream of sorted records into an
// archive chunk file.
type chunkWriter struct {
	ark             *Archive
	chunk           Chunk
	dataFileWriter  *zngio.Writer
	indexBuilder    *zng.Builder
	indexTempPath   string
	indexTempWriter *zngio.Writer
	needIndexWrite  bool
	tsDir           tsDir
}

func newChunkWriter(ctx context.Context, ark *Archive, tsDir tsDir, chunk Chunk) (*chunkWriter, error) {
	out, err := ark.dataSrc.NewWriter(ctx, chunk.Path(ark))
	if err != nil {
		return nil, err
	}
	dataFileWriter := zngio.NewWriter(bufwriter.New(out), zngio.WriterOpts{
		LZ4BlockSize:     importLZ4BlockSize,
		StreamRecordsMax: importStreamRecordsMax,
	})
	// Create the temporary index key file
	idxTemp, err := ioutil.TempFile("", "archive-import-index-key-")
	if err != nil {
		return nil, err
	}
	indexTempPath := idxTemp.Name()
	indexTempWriter := zngio.NewWriter(bufwriter.New(idxTemp), zngio.WriterOpts{})
	zctx := resolver.NewContext()
	indexBuilder := zng.NewBuilder(zctx.MustLookupTypeRecord([]zng.Column{
		{"ts", zng.TypeTime},
		{"offset", zng.TypeInt64},
	}))
	return &chunkWriter{
		ark:             ark,
		chunk:           chunk,
		dataFileWriter:  dataFileWriter,
		indexBuilder:    indexBuilder,
		indexTempPath:   indexTempPath,
		indexTempWriter: indexTempWriter,
		needIndexWrite:  true,
		tsDir:           tsDir,
	}, nil
}

func (cw *chunkWriter) Write(rec *zng.Record) error {
	// We want to index the start of stream (SOS) position of the data file by
	// record timestamps; we don't know when we've started a new stream until
	// after we written the first record in the stream.
	sos := cw.dataFileWriter.LastSOS()
	if err := cw.dataFileWriter.Write(rec); err != nil {
		return err
	}
	if cw.needIndexWrite {
		out := cw.indexBuilder.Build(zng.EncodeTime(rec.Ts()), zng.EncodeInt(sos))
		if err := cw.indexTempWriter.Write(out); err != nil {
			return err
		}
		cw.needIndexWrite = false
	}
	if cw.dataFileWriter.LastSOS() != sos {
		cw.needIndexWrite = true
	}
	return nil
}

// abort should be called when an error occurs during write. Errors are ignored
// because the write error will be more informative and should be returned.
func (cw *chunkWriter) abort() {
	cw.dataFileWriter.Close()
	cw.indexTempWriter.Close()
	os.Remove(cw.indexTempPath)
}

func (cw *chunkWriter) close(ctx context.Context) error {
	err := cw.dataFileWriter.Close()
	if closeErr := cw.indexTempWriter.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	err = writeChunkMetadata(ctx, chunkMetadataPath(cw.ark, cw.tsDir, cw.chunk.Id), chunkMetadata{
		First:       cw.chunk.First,
		Last:        cw.chunk.Last,
		Kind:        FileKindData,
		RecordCount: cw.chunk.RecordCount,
	})
	if err != nil {
		return err
	}
	// Write the time seek index into the archive, feeding it the key/offset
	// records written to indexTempPath.
	tf, err := fs.Open(cw.indexTempPath)
	if err != nil {
		return err
	}
	defer func() {
		tf.Close()
		os.Remove(tf.Name())
	}()
	zctx := resolver.NewContext()
	tfr := zngio.NewReader(tf, zctx)
	idxURI := cw.chunk.seekIndexPath(cw.ark)
	idxWriter, err := microindex.NewWriter(zctx, idxURI.String(), []field.Static{field.New("ts")}, framesize)
	if err != nil {
		return err
	}
	// XXX: zq#1329
	// The microindex finder doesn't yet handle searching when keys
	// are stored in descending order, which is the zar default.
	if err = zbuf.CopyWithContext(ctx, idxWriter, tfr); err != nil {
		idxWriter.Abort()
		return err
	}
	// TODO: zq#1264
	// Add an entry to the update log for S3 backed stores containing the
	// location of the just added data & index file.
	return idxWriter.Close()
}

func importCompareFn(ark *Archive) expr.CompareFn {
	return func(a, b *zng.Record) (cmp int) {
		d := a.Ts() - b.Ts()
		if d < 0 {
			cmp = -1
		}
		if d > 0 {
			cmp = 1
		}
		return cmp * ark.DataOrder.Int()
	}
}
