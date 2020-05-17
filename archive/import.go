package archive

import (
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
)

func tsDir(ts nano.Ts) string {
	return ts.Time().Format("20060102")
}

func Import(ark *Archive, r zbuf.Reader) error {
	var w *bufwriter.Writer
	var zw zbuf.Writer
	var n int
	for {
		rec, err := r.Read()
		if err != nil || rec == nil {
			if w != nil {
				if err := w.Close(); err != nil {
					return err
				}
			}
			return err
		}
		if w == nil {
			ts := rec.Ts
			dir := filepath.Join(ark.Root, tsDir(ts))
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
			w = bufwriter.New(out)
			zw = zngio.NewWriter(w, zio.WriterFlags{})
		}
		if err := zw.Write(rec); err != nil {
			return err
		}
		n += len(rec.Raw)
		if int64(n) >= ark.Config.LogSizeThreshold {
			if err := w.Close(); err != nil {
				return err
			}
			w = nil
			n = 0
		}
	}
}
