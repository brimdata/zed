package ingest

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

// logdriver implements driver.Driver.
type logdriver struct {
	pipe      *api.JSONPipe
	startTime nano.Ts
	writers   []zbuf.Writer
}

func (d *logdriver) Write(cid int, batch zbuf.Batch) error {
	if len(d.writers) == 1 {
		cid = 0
	}
	for _, r := range batch.Records() {
		if err := d.writers[cid].Write(r); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (d *logdriver) Warn(warning string) error {
	return d.pipe.Send(&api.LogPostWarning{
		Type: "LogPostWarning",
		Msg:  warning,
	})
}
func (d *logdriver) Stats(stats api.ScannerStats) error {
	return nil
}

func (d *logdriver) ChannelEnd(cid int, stats api.ScannerStats) error {
	return d.Stats(stats)
}
