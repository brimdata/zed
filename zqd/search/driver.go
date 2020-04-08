package search

import (
	"context"
	"errors"
	"time"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

type driver interface {
	Warn(msg string) error
	Write(int, zbuf.Batch) error
	ChannelEnd(int, api.ScannerStats) error
	Stats(api.ScannerStats) error
}

func run(out *proc.MuxOutput, d driver, statsInterval time.Duration) error {
	//stats are zero at this point.
	var stats api.ScannerStats
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()
	for !out.Complete() {
		chunk := out.Pull(ticker.C)
		if chunk.Err != nil {
			if chunk.Err == proc.ErrTimeout {
				/* not yet
				err := d.sendStats(out.Stats())
				if err != nil {
					return d.abort(0, err)
				}
				*/
				continue
			}
			if chunk.Err == context.Canceled {
				out.Drain()
				return errors.New("canceled")
			}
			return chunk.Err
		}
		if chunk.Warning != "" {
			if err := d.Warn(chunk.Warning); err != nil {
				return err
			}
		}
		if chunk.Batch == nil {
			// One of the flowgraph tails is done.  We send stats and
			// a done message for each channel that finishes
			if err := d.ChannelEnd(chunk.ID, stats); err != nil {
				return err
			}
		} else {
			if err := d.Write(chunk.ID, chunk.Batch); err != nil {
				return err
			}
		}
	}
	return nil
}
