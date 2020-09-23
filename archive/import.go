package archive

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zql"
)

// The below are vars for unit testing.
var (
	importLZ4BlockSize     = zngio.DefaultLZ4BlockSize
	importStreamRecordsMax = zngio.DefaultStreamRecordsMax
)

type importDriver struct {
	ark             *Archive
	ctx             context.Context
	dataFile        dataFile
	dataFileWriter  *zngio.Writer
	firstTs         nano.Ts
	indexBuilder    *zng.Builder
	indexTempPath   string
	indexTempWriter *zngio.Writer
	lastTs          nano.Ts
	needIndexWrite  bool
	rcount          int64
	tsDir           iosrc.URI
	zctx            *resolver.Context
}

func (d *importDriver) newWriter(rec *zng.Record) error {
	d.firstTs = rec.Ts()
	d.rcount = 0

	// Create the data file writer
	d.tsDir = d.ark.DataPath.AppendPath(dataDirname, newTsDir(d.firstTs).name())
	if dirmkr, ok := d.ark.dataSrc.(iosrc.DirMaker); ok {
		if err := dirmkr.MkdirAll(d.tsDir, 0755); err != nil {
			return err
		}
	}
	d.dataFile = newDataFile()
	out, err := d.ark.dataSrc.NewWriter(d.ctx, d.tsDir.AppendPath(d.dataFile.name()))
	if err != nil {
		return err
	}
	d.dataFileWriter = zngio.NewWriter(bufwriter.New(out), zngio.WriterOpts{
		LZ4BlockSize:     importLZ4BlockSize,
		StreamRecordsMax: importStreamRecordsMax,
	})

	// Create the temporary index key file
	idxTemp, err := ioutil.TempFile("", "archive-import-index-key-")
	if err != nil {
		return err
	}
	d.indexTempPath = idxTemp.Name()
	d.indexTempWriter = zngio.NewWriter(bufwriter.New(idxTemp), zngio.WriterOpts{})
	if d.indexBuilder == nil {
		d.zctx = resolver.NewContext()
		d.indexBuilder = zng.NewBuilder(d.zctx.MustLookupTypeRecord([]zng.Column{
			{"ts", zng.TypeTime},
			{"offset", zng.TypeInt64},
		}))
	}
	return nil
}

func (d *importDriver) writeOne(rec *zng.Record) error {
	if d.dataFileWriter != nil && !d.firstTs.DayOf().Contains(rec.Ts()) {
		// Don't allow data files to include records from multiple tsDir time spans.
		if err := d.close(); err != nil {
			return err
		}
	}
	if d.dataFileWriter == nil {
		if err := d.newWriter(rec); err != nil {
			return err
		}
	}

	// We want to index the start of stream (SOS) position of the data file by
	// record timestamps; we don't know when we've started a new stream until
	// after we written the first record in the stream.
	sos := d.dataFileWriter.LastSOS()
	if err := d.dataFileWriter.Write(rec); err != nil {
		return err
	}
	d.lastTs = rec.Ts()
	d.rcount++
	if d.needIndexWrite {
		out := d.indexBuilder.Build(zng.EncodeTime(d.lastTs), zng.EncodeInt(sos))
		if err := d.indexTempWriter.Write(out); err != nil {
			return err
		}
		d.needIndexWrite = false
	}
	if d.dataFileWriter.LastSOS() != sos {
		d.needIndexWrite = true
	}
	if d.dataFileWriter.Position() >= d.ark.LogSizeThreshold {
		if err := d.close(); err != nil {
			return err
		}
	}
	return nil
}

func (d *importDriver) close() error {
	if d.dataFileWriter == nil {
		return nil
	}
	err := d.dataFileWriter.Close()
	if closeErr := d.indexTempWriter.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	d.dataFileWriter = nil

	// Write the time seek index into the archive, feeding it the key/offset
	// records written to indexTempPath.
	tf, err := fs.Open(d.indexTempPath)
	if err != nil {
		return err
	}
	defer func() {
		tf.Close()
		os.Remove(tf.Name())
	}()
	tfr := zngio.NewReader(tf, d.zctx)
	sf := seekIndexFile{id: d.dataFile.id, recordCount: d.rcount, first: d.firstTs, last: d.lastTs}
	idxURI := d.tsDir.AppendPath(sf.name())
	idxWriter, err := microindex.NewWriter(d.zctx, idxURI.String(), []string{"ts"}, framesize)
	if err != nil {
		return err
	}
	// XXX: zq#1329
	// The microindex finder doesn't yet handle searching when keys
	// are stored in descending order, which is the zar default.
	if err = zbuf.CopyWithContext(d.ctx, idxWriter, tfr); err != nil {
		idxWriter.Abort()
		return err
	}
	// TODO: zq#1264
	// Add an entry to the update log for S3 backed stores containing the
	// location of the just added data & index file.
	return idxWriter.Close()
}

func (d *importDriver) Write(cid int, batch zbuf.Batch) error {
	if cid != 0 {
		panic("importDriver write to non-zero channel")
	}
	for i := 0; i < batch.Length(); i++ {
		if err := d.writeOne(batch.Index(i)); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (d *importDriver) ChannelEnd(cid int) error {
	if cid != 0 {
		panic("importDriver ChannelEnd to non-zero channel")
	}
	return d.close()
}

func (d *importDriver) Warn(warning string) error          { return nil }
func (d *importDriver) Stats(stats api.ScannerStats) error { return nil }

func importProc(ark *Archive) string {
	if ark.DataSortDirection == zbuf.DirTimeForward {
		return "sort ts"
	}
	return "sort -r ts"
}

func Import(ctx context.Context, ark *Archive, zctx *resolver.Context, r zbuf.Reader) error {
	proc, err := zql.ParseProc(importProc(ark))
	if err != nil {
		return err
	}
	id := &importDriver{
		ark:            ark,
		ctx:            ctx,
		needIndexWrite: true,
	}
	if err := driver.Run(ctx, id, proc, zctx, r, driver.Config{}); err != nil {
		return fmt.Errorf("archive.Import: run failed: %w", err)
	}
	return nil
}
