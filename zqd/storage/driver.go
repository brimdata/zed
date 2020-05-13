package storage

import (
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

// zngdriver implements driver.Driver.
type zngdriver struct {
	w zbuf.Writer
}

func (s *zngdriver) Write(_ int, batch zbuf.Batch) error {
	for i := 0; i < batch.Length(); i++ {
		if err := s.w.Write(batch.Index(i)); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (s *zngdriver) Warn(warning string) error          { return nil }
func (s *zngdriver) Stats(stats api.ScannerStats) error { return nil }
func (s *zngdriver) ChannelEnd(cid int) error           { return nil }
