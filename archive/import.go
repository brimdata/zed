package archive

import (
	"context"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zql"
	"go.uber.org/zap"
)

func tsDir(ts nano.Ts) string {
	return ts.Time().Format("20060102")
}

type importDriver struct {
	ark *Archive
	bw  *bufwriter.Writer
	zw  zbuf.Writer
	n   int64
}

func (d *importDriver) writeOne(rec *zng.Record) error {
	if d.zw == nil {
		ts := rec.Ts
		dir := filepath.Join(d.ark.Root, tsDir(ts))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		path := filepath.Join(dir, ts.StringFloat()+".zng")
		//XXX for now just truncate any existing file.
		// a future PR will do a split/merge.
		out, err := fs.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		d.bw = bufwriter.New(out)
		d.zw = zngio.NewWriter(d.bw, zio.WriterFlags{})
	}
	if err := d.zw.Write(rec); err != nil {
		return err
	}
	d.n += int64(len(rec.Raw))
	if d.n >= d.ark.Config.LogSizeThreshold {
		if err := d.close(); err != nil {
			return err
		}
	}
	return nil
}

func (d *importDriver) close() error {
	if d.bw != nil {
		if err := d.bw.Close(); err != nil {
			return err
		}
	}
	d.zw = nil
	d.n = 0
	return nil
}

func (d *importDriver) Write(_ int, batch zbuf.Batch) error {
	for i := 0; i < batch.Length(); i++ {
		if err := d.writeOne(batch.Index(i)); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (d *importDriver) ChannelEnd(cid int) error {
	return d.close()
}

func (d *importDriver) Warn(warning string) error          { return nil }
func (d *importDriver) Stats(stats api.ScannerStats) error { return nil }

func importProc(ark *Archive) string {
	if ark.Config.DataSortForward {
		return "sort ts"
	} else {
		return "sort -r ts"
	}
}

func Import(ark *Archive, r zbuf.Reader) error {
	proc, err := zql.ParseProc(importProc(ark))
	if err != nil {
		return err
	}

	fg, err := driver.Compile(context.TODO(), proc, r, false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}

	return driver.Run(fg, &importDriver{ark: ark}, nil)
}
