package ingest

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

// logdriver implements driver.Driver.
type logdriver struct {
	pipe         *api.JSONPipe
	startTime    nano.Ts
	totalSize    int64
	lastReadSize int64
	writers      []zbuf.Writer
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
		Type:    "LogPostWarning",
		Warning: warning,
	})
}

func (d *logdriver) Stats(stats api.ScannerStats) error {
	d.lastReadSize = stats.BytesRead
	return d.pipe.Send(&api.LogPostStatus{
		Type:         "LogPostStatus",
		LogTotalSize: d.totalSize,
		LogReadSize:  d.lastReadSize,
	})
}

func (d *logdriver) ChannelEnd(cid int) error {
	return nil
}

// simpledriver implements driver.Driver.
type simpledriver struct {
	w zbuf.Writer
}

func (s *simpledriver) Write(_ int, batch zbuf.Batch) error {
	for _, r := range batch.Records() {
		if err := s.w.Write(r); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (s *simpledriver) Warn(warning string) error {
	return nil
}
func (s *simpledriver) Stats(stats api.ScannerStats) error {
	return nil
}

func (s *simpledriver) ChannelEnd(cid int) error {
	return nil
}
