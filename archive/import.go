package archive

import (
	"context"
	"fmt"
	"path"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zql"
)

func tsDir(ts nano.Ts) string {
	return ts.Time().Format("20060102")
}

type importDriver struct {
	ark *Archive
	zw  *zngio.Writer

	span   nano.Span
	logID  LogID
	spans  []SpanInfo
	rcount int
}

func (d *importDriver) writeOne(rec *zng.Record) error {
	recspan := nano.Span{rec.Ts(), 1}
	if d.zw == nil {
		dname := tsDir(rec.Ts())
		fname := rec.Ts().StringFloat() + ".zng"
		d.span = recspan
		d.rcount = 0
		// Create LogID with path.Join so that it always uses forward
		// slashes (dir1/foo.zng), regardless of platform.
		d.logID = LogID(path.Join(dname, fname))

		dpath := d.ark.DataPath.AppendPath(dname)
		if dirmkr, ok := d.ark.dataSrc.(iosrc.DirMaker); ok {
			if err := dirmkr.MkdirAll(dpath, 0755); err != nil {
				return err
			}
		}

		//XXX for now just truncate any existing file.
		// a future PR will do a split/merge.
		fpath := dpath.AppendPath(fname)
		out, err := d.ark.dataSrc.NewWriter(fpath)
		if err != nil {
			return err
		}
		bw := bufwriter.New(out)
		d.zw = zngio.NewWriter(bw, zngio.WriterOpts{LZ4BlockSize: zngio.DefaultLZ4BlockSize})
	} else {
		d.span = d.span.Union(recspan)
	}
	if err := d.zw.Write(rec); err != nil {
		return err
	}
	d.rcount++
	if d.zw.Position() >= d.ark.LogSizeThreshold {
		if err := d.close(); err != nil {
			return err
		}
	}
	return nil
}

func (d *importDriver) close() error {
	if d.zw != nil {
		if err := d.zw.Close(); err != nil {
			return err
		}
		d.spans = append(d.spans, SpanInfo{
			Span:        d.span,
			LogID:       d.logID,
			RecordCount: d.rcount,
		})
		d.zw = nil
	}
	return nil
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

	id := &importDriver{ark: ark}
	if err := driver.Run(ctx, id, proc, zctx, r, driver.Config{}); err != nil {
		return fmt.Errorf("archive.Import: run failed: %w", err)
	}

	return ark.AppendSpans(id.spans)
}
