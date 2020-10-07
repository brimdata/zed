package archive

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/expr"
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
	err1 := zbuf.CopyWithContext(ctx, w, r)
	err2 := w.close()
	if err1 != nil {
		return err1
	}
	return err2
}

// importWriter is a zbuf.Writer the partitions records by day into the
// appropriate tsDirWriter. importWriter keeps track of the overall memory
// footprint of the collection of tsDirWriter and instructs the tsDirWriter
// with the largest footprint to spill its records to a temporary file on disk.
//
// XXX importWriter does not currently keep track of size of records written
// to temporary files. At some point this should have a maxTempFileSize to
// ensure the importWriter does not exceed the size of a provisioned tmpfs.
type importWriter struct {
	ark     *Archive
	ctx     context.Context
	writers map[nano.Ts]*tsDirWriter

	bufSize int64
}

func newImportWriter(ctx context.Context, ark *Archive) *importWriter {
	return &importWriter{
		ark:     ark,
		ctx:     ctx,
		writers: make(map[nano.Ts]*tsDirWriter),
	}
}

func (w *importWriter) Write(rec *zng.Record) error {
	day := rec.Ts().Midnight()
	dw, ok := w.writers[day]
	if !ok {
		var err error
		dw, err = newTsDirWriter(w, day)
		if err != nil {
			return err
		}
		w.writers[day] = dw
	}
	if err := dw.writeOne(rec); err != nil {
		return err
	}
	for w.bufSize > ImportBufSize {
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
func (w *importWriter) spillLargestBuffer() error {
	var dw *tsDirWriter
	for _, w := range w.writers {
		if dw == nil || w.size > dw.size {
			dw = w
		}
	}
	return dw.spill()
}

func (i *importWriter) close() error {
	var merr error
	for ts, w := range i.writers {
		if err := w.flush(); err != nil {
			merr = multierr.Append(merr, err)
		}
		delete(i.writers, ts)
	}
	return merr
}

// tsDirWriter accumlates to records to be written to a particular tsDir-
// currently segmented by day. When a tsdir receives enough records to exceed
// ark.LogSizeThreshold, the underlying records are written to a chunk file in
// the archive.
type tsDirWriter struct {
	importWriter *importWriter
	ark          *Archive
	ctx          context.Context
	day          nano.Ts
	records      []*zng.Record
	size         int64
	spiller      *spill.MergeSort
	tsDir        iosrc.URI
}

func newTsDirWriter(iw *importWriter, midnight nano.Ts) (*tsDirWriter, error) {
	d := &tsDirWriter{
		ark:          iw.ark,
		importWriter: iw,
		ctx:          iw.ctx,
		day:          midnight,
		tsDir:        iw.ark.DataPath.AppendPath(dataDirname, newTsDir(midnight).name()),
	}
	if dirmkr, ok := d.ark.dataSrc.(iosrc.DirMaker); ok {
		if err := dirmkr.MkdirAll(d.tsDir, 0755); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (w *tsDirWriter) addBufSize(delta int64) {
	old := w.bufSize()
	w.size += delta
	w.importWriter.bufSize += w.bufSize() - old
}

// bufSize returns the actually buffer size as a multiple of importLZ4BlockSize.
// This is done to ensure that whenever spill the generated spill is compressed
// nicely onto disk.
func (w *tsDirWriter) bufSize() int64 {
	return (w.size / int64(importLZ4BlockSize)) * int64(importLZ4BlockSize)
}

// approxTotalBytes is the sum of the size of compressed records spilt to disk
// and a crude approximation of the buffer record bytes (simply bufBytes / 2).
func (d *tsDirWriter) approxTotalBytes() int64 {
	b := (d.bufSize() / 2)
	if d.spiller != nil {
		b += d.spiller.SpillSize()
	}
	return b
}

func (d *tsDirWriter) writeOne(rec *zng.Record) error {
	d.records = append(d.records, rec)
	d.addBufSize(int64(len(rec.Raw)))
	if d.approxTotalBytes() > d.ark.LogSizeThreshold {
		if err := d.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (d *tsDirWriter) spill() error {
	if len(d.records) == 0 {
		return nil
	}
	if d.spiller == nil {
		var err error
		d.spiller, err = spill.NewMergeSort(importCompareFn(d.ark))
		if err != nil {
			return err
		}
		d.spiller.SetWriterOpts(zngio.WriterOpts{
			LZ4BlockSize:     importLZ4BlockSize,
			StreamRecordsMax: importStreamRecordsMax,
		})
	}
	if err := d.spiller.Spill(d.records); err != nil {
		return err
	}
	d.records = d.records[:0]
	d.addBufSize(d.size * -1)
	return nil
}

func (d *tsDirWriter) flush() error {
	var r zbuf.Reader
	if d.spiller != nil {
		if err := d.spill(); err != nil {
			return err
		}
		spiller := d.spiller
		d.spiller, r = nil, spiller
		defer spiller.Cleanup()
	} else {
		// If len of records is 0 and spiller is nil, the tsDirWriter is empty.
		// Just return nil.
		if len(d.records) == 0 {
			return nil
		}
		expr.SortStable(d.records, importCompareFn(d.ark))
		r = zbuf.Array(d.records).NewReader()
		defer func() {
			d.records = d.records[:0]
			d.addBufSize(d.size * -1)
		}()
	}
	w, err := newChunkWriter(d.ctx, d.ark, d.tsDir)
	if err != nil {
		return err
	}
	if err := zbuf.CopyWithContext(d.ctx, w, r); err != nil {
		w.close(d.ctx)
		return err
	}
	if err := w.close(d.ctx); err != nil {
		return err
	}
	return nil
}

// chunkWriter is a zbuf.Writer that writes a stream of sorted records into an
// archive chunk file. chunkWriter is created and written to  by tsDirWriter
// when it recieves the tsDirWriter.flush() call.
type chunkWriter struct {
	dataFile        dataFile
	dataFileWriter  *zngio.Writer
	indexBuilder    *zng.Builder
	indexTempPath   string
	indexTempWriter *zngio.Writer

	tsDir           iosrc.URI
	firstTs, lastTs nano.Ts
	rcount          int
	needIndexWrite  bool
}

func newChunkWriter(ctx context.Context, ark *Archive, tsDir iosrc.URI) (*chunkWriter, error) {
	dataFile := newDataFile()
	out, err := ark.dataSrc.NewWriter(ctx, tsDir.AppendPath(dataFile.name()))
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
		dataFile:        dataFile,
		dataFileWriter:  dataFileWriter,
		indexBuilder:    indexBuilder,
		indexTempPath:   indexTempPath,
		indexTempWriter: indexTempWriter,
		tsDir:           tsDir,
		needIndexWrite:  true,
	}, nil
}

func (c *chunkWriter) Write(rec *zng.Record) error {
	// We want to index the start of stream (SOS) position of the data file by
	// record timestamps; we don't know when we've started a new stream until
	// after we written the first record in the stream.
	sos := c.dataFileWriter.LastSOS()
	if err := c.dataFileWriter.Write(rec); err != nil {
		return err
	}
	c.lastTs = rec.Ts()
	if c.firstTs == 0 {
		c.firstTs = c.lastTs
	}
	c.rcount++
	if c.needIndexWrite {
		out := c.indexBuilder.Build(zng.EncodeTime(c.lastTs), zng.EncodeInt(sos))
		if err := c.indexTempWriter.Write(out); err != nil {
			return err
		}
		c.needIndexWrite = false
	}
	if c.dataFileWriter.LastSOS() != sos {
		c.needIndexWrite = true
	}
	return nil
}

func (c *chunkWriter) close(ctx context.Context) error {
	err := c.dataFileWriter.Close()
	if closeErr := c.indexTempWriter.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	c.dataFileWriter = nil
	// Write the time seek index into the archive, feeding it the key/offset
	// records written to indexTempPath.
	tf, err := fs.Open(c.indexTempPath)
	if err != nil {
		return err
	}
	defer func() {
		tf.Close()
		os.Remove(tf.Name())
	}()
	tfr := zngio.NewReader(tf, resolver.NewContext())
	sf := seekIndexFile{id: c.dataFile.id, recordCount: c.rcount, first: c.firstTs, last: c.lastTs}
	idxURI := c.tsDir.AppendPath(sf.name())
	idxWriter, err := microindex.NewWriter(resolver.NewContext(), idxURI.String(), []string{"ts"}, framesize)
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
		return cmp * ark.DataSortDirection.Int()
	}
}
