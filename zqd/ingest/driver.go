package ingest

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

// driver implements search.Driver
type driver struct {
	pipe      *api.JSONPipe
	startTime nano.Ts
	writers   []zbuf.Writer
}

func (d *driver) Write(cid int, arr zbuf.Batch) error {
	if len(d.writers) == 1 {
		cid = 0
	}
	for _, r := range arr.Records() {
		if err := d.writers[cid].Write(r); err != nil {
			return err
		}
	}
	arr.Unref()
	return nil
}

func (d *driver) Warn(warning string) error {
	return nil
}
func (d *driver) Stats(stats api.ScannerStats) error {
	return nil
}

func (d *driver) ChannelEnd(cid int, stats api.ScannerStats) error {
	if err := d.Stats(stats); err != nil {
		return err
	}
	return nil
}
