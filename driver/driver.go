package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/multisource"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
)

type Driver interface {
	Warn(msg string) error
	Write(channelID int, batch zbuf.Batch) error
	ChannelEnd(channelID int) error
	Stats(api.ScannerStats) error
}

func Run(ctx context.Context, d Driver, program ast.Proc, zctx *resolver.Context, reader zbuf.Reader, cfg Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux, err := compileSingle(ctx, program, zctx, reader, cfg)
	if err != nil {
		return err
	}
	return runMux(mux, d, cfg.StatsTick)
}

func MultiRun(ctx context.Context, d Driver, program ast.Proc, zctx *resolver.Context, msrc multisource.MultiSource, mcfg MultiConfig) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux, err := compileMulti(ctx, program, zctx, msrc, mcfg)
	if err != nil {
		return err
	}
	return runMux(mux, d, mcfg.StatsTick)
}

func runMux(out *muxOutput, d Driver, statsTickCh <-chan time.Time) error {
	for !out.Complete() {
		chunk := out.Pull(statsTickCh)
		if chunk.Err != nil {
			if chunk.Err == errTimeout {
				if err := d.Stats(out.Stats()); err != nil {
					return err
				}
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
			// One of the flowgraph tails is done.
			if err := d.ChannelEnd(chunk.ID); err != nil {
				return err
			}
		} else {
			if err := d.Write(chunk.ID, chunk.Batch); err != nil {
				return err
			}
		}
	}
	if statsTickCh != nil {
		return d.Stats(out.Stats())
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
	batch.Unref()
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

func (d *CLI) ChannelEnd(int) error         { return nil }
func (d *CLI) Stats(api.ScannerStats) error { return nil }

type transformDriver struct {
	w zbuf.Writer
}

func (d *transformDriver) Write(cid int, batch zbuf.Batch) error {
	if cid != 0 {
		return errors.New("transform proc with multiple tails")
	}
	for i := 0; i < batch.Length(); i++ {
		if err := d.w.Write(batch.Index(i)); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (d *transformDriver) Warn(warning string) error          { return nil }
func (d *transformDriver) Stats(stats api.ScannerStats) error { return nil }
func (d *transformDriver) ChannelEnd(cid int) error           { return nil }

// Copy applies a proc to all records from a zbuf.Reader, writing to a
// single zbuf.Writer. The proc must have a single tail.
func Copy(ctx context.Context, w zbuf.Writer, prog ast.Proc, zctx *resolver.Context, r zbuf.Reader, cfg Config) error {
	d := &transformDriver{w: w}
	return Run(ctx, d, prog, zctx, r, cfg)
}
