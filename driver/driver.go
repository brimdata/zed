package driver

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

type Driver interface {
	Warn(msg string) error
	Write(channelID int, batch zbuf.Batch) error
	ChannelEnd(channelID int, stats api.ScannerStats) error
	Stats(api.ScannerStats) error
}

func Run(out *MuxOutput, d Driver, statsTickCh <-chan time.Time) error {
	//stats are zero at this point.
	var stats api.ScannerStats
	for !out.Complete() {
		chunk := out.Pull(statsTickCh)
		if chunk.Err != nil {
			if chunk.Err == ErrTimeout {
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

// CLI implements Driver.
type CLI struct {
	writers  []zbuf.Writer
	warnings io.Writer
}

func NewCLI(w ...zbuf.Writer) *CLI {
	return &CLI{
		writers: w,
	}
}

func (d *CLI) SetWarningsWriter(w io.Writer) {
	d.warnings = w
}

func (d *CLI) Write(cid int, batch zbuf.Batch) error {
	if len(d.writers) == 1 {
		cid = 0
	}
	for _, r := range batch.Records() {
		if err := d.writers[cid].Write(r); err != nil {
			return err
		}
	}
	return nil
}

func (d *CLI) Warn(msg string) error {
	if d.warnings != nil {
		if _, err := fmt.Fprintln(d.warnings, msg); err != nil {
			return err
		}
	}
	return nil
}

func (d *CLI) ChannelEnd(int, api.ScannerStats) error { return nil }
func (d *CLI) Stats(api.ScannerStats) error           { return nil }
