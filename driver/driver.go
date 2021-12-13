package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/mux"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zqe"
	"go.uber.org/zap"
)

type Driver interface {
	Warn(msg string) error
	Write(channelID int, batch zbuf.Batch) error
	ChannelEnd(channelID int) error
	Stats(zbuf.ScannerStats) error
}

func RunWithReader(ctx context.Context, d Driver, program ast.Proc, zctx *zed.Context, reader zio.Reader, logger *zap.Logger) error {
	pctx := proc.NewContext(ctx, zctx, logger)
	runtime, err := compiler.CompileForInternal(pctx, program, reader)
	if err != nil {
		pctx.Cancel()
		return err
	}
	return run(pctx, d, runtime, nil)
}

func RunWithOrderedReader(ctx context.Context, d Driver, program ast.Proc, zctx *zed.Context, reader zio.Reader, layout order.Layout, logger *zap.Logger) error {
	pctx := proc.NewContext(ctx, zctx, logger)
	runtime, err := compiler.CompileForInternalWithOrder(pctx, program, reader, layout)
	if err != nil {
		pctx.Cancel()
		return err
	}
	return run(pctx, d, runtime, nil)
}

func RunWithFileSystem(ctx context.Context, d Driver, program ast.Proc, zctx *zed.Context, readers []zio.Reader, adaptor proc.DataAdaptor) (zbuf.ScannerStats, error) {
	pctx := proc.NewContext(ctx, zctx, nil)
	runtime, err := compiler.CompileForFileSystem(pctx, program, readers, adaptor)
	if err != nil {
		pctx.Cancel()
		return zbuf.ScannerStats{}, err
	}
	err = run(pctx, d, runtime, nil)
	return runtime.Statser().Stats(), err
}

func RunWithLake(ctx context.Context, d Driver, program ast.Proc, zctx *zed.Context, lake proc.DataAdaptor, head *lakeparse.Commitish) (zbuf.ScannerStats, error) {
	pctx := proc.NewContext(ctx, zctx, nil)
	runtime, err := compiler.CompileForLake(pctx, program, lake, 0, head)
	if err != nil {
		pctx.Cancel()
		return zbuf.ScannerStats{}, err
	}
	err = run(pctx, d, runtime, nil)
	return runtime.Statser().Stats(), err
}

func RunWithLakeAndStats(ctx context.Context, d Driver, program ast.Proc, zctx *zed.Context, lake proc.DataAdaptor, head *lakeparse.Commitish, ticker <-chan time.Time, logger *zap.Logger, parallelism int) error {
	pctx := proc.NewContext(ctx, zctx, logger)
	runtime, err := compiler.CompileForLake(pctx, program, lake, parallelism, head)
	if err != nil {
		pctx.Cancel()
		return err
	}
	return run(pctx, d, runtime, ticker)
}

func run(pctx *proc.Context, d Driver, runtime *compiler.Runtime, statsTicker <-chan time.Time) error {
	puller := runtime.Puller()
	if puller == nil {
		return errors.New("internal error: driver called with unprepared runtime")
	}
	statser := runtime.Statser()
	if statser == nil && statsTicker != nil {
		return errors.New("internal error: driver wants live stats but runtime has no statser")
	}
	resultCh := make(chan proc.Result)
	go func() {
		var eof bool
		for {
			batch, err := safePull(puller)
			if batch == nil || err != nil {
				if eof || err != nil {
					if err != nil {
						resultCh <- proc.Result{Err: err}
					}
					close(resultCh)
					return
				}
				eof = true
				continue
			}
			eof = false
			resultCh <- proc.Result{Batch: batch}
		}
	}()
	defer func() {
		pctx.Cancel()
		// Drain resultCh so puller sees cancellation and can clean up.
		for {
			if _, ok := <-resultCh; !ok {
				return
			}
		}
	}()
	for {
		select {
		case <-statsTicker:
			if err := d.Stats(statser.Stats()); err != nil {
				return err
			}
		case result := <-resultCh:
			if result.Batch == nil || result.Err != nil {
				err := result.Err
				// Now that we're done, drain the warnings.
				// This is a little goofy and we should clean
				// up the warnings model with its own package.
				// See issue #2600.
				go func() {
					close(pctx.Warnings)
				}()
				for warning := range pctx.Warnings {
					if warnErr := d.Warn(warning); err == nil {
						err = warnErr
					}
				}
				if len(runtime.Outputs()) == 1 {
					if endErr := d.ChannelEnd(0); err == nil {
						err = endErr
					}
				}
				if statser != nil {
					if statsErr := d.Stats(statser.Stats()); err == nil {
						err = statsErr
					}
				}
				return err
			}
			batch, cid := extractLabel(result.Batch)
			if batch == nil {
				if err := d.ChannelEnd(cid); err != nil {
					return err
				}
			} else {
				if err := d.Write(cid, batch); err != nil {
					return err
				}
			}
		case warning := <-pctx.Warnings:
			if err := d.Warn(warning); err != nil {
				return err
			}
		case <-pctx.Done():
			return pctx.Err()
		}
	}
}

func extractLabel(p zbuf.Batch) (zbuf.Batch, int) {
	var label int
	if labeled, ok := p.(*mux.Batch); ok {
		label = labeled.Label
		p = labeled.Batch
	}
	return p, label
}

func safePull(puller zbuf.Puller) (b zbuf.Batch, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err = zqe.RecoverError(r)
	}()
	b, err = puller.Pull()
	return
}

// CLI implements Driver.
type CLI struct {
	writers  []zio.Writer
	warnings io.Writer
}

func NewCLI(w ...zio.Writer) *CLI {
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
	defer batch.Unref()
	return zbuf.WriteBatch(d.writers[cid], batch)
}

func (d *CLI) Warn(msg string) error {
	if d.warnings != nil {
		if _, err := fmt.Fprintln(d.warnings, msg); err != nil {
			return err
		}
	}
	return nil
}

func (d *CLI) ChannelEnd(int) error          { return nil }
func (d *CLI) Stats(zbuf.ScannerStats) error { return nil }

type transformDriver struct {
	w zio.Writer
}

func (d *transformDriver) Write(cid int, batch zbuf.Batch) error {
	if cid != 0 {
		return errors.New("transform proc with multiple tails")
	}
	defer batch.Unref()
	return zbuf.WriteBatch(d.w, batch)
}

func (d *transformDriver) Warn(warning string) error           { return nil }
func (d *transformDriver) Stats(stats zbuf.ScannerStats) error { return nil }
func (d *transformDriver) ChannelEnd(cid int) error            { return nil }

// Copy applies a proc to all records from a zio.Reader, writing to a
// single zio.Writer. The proc must have a single tail.
func Copy(ctx context.Context, w zio.Writer, prog ast.Proc, zctx *zed.Context, r zio.Reader, logger *zap.Logger) error {
	d := &transformDriver{w: w}
	return RunWithReader(ctx, d, prog, zctx, r, logger)
}

// Reader implements zio.ReaderCloser and Driver.
type Reader struct {
	io.Closer
	runtime *compiler.Runtime
	once    sync.Once
	batch   zbuf.Batch
	index   int
	batchCh chan zbuf.Batch
	// err protected by batchCh
	err error
}

func (*Reader) Warn(warning string) error           { return nil }
func (*Reader) Stats(stats zbuf.ScannerStats) error { return nil }
func (*Reader) ChannelEnd(cid int) error            { return nil }

func (r *Reader) Write(_ int, batch zbuf.Batch) error {
	if batch != nil {
		r.batchCh <- batch
	}
	return nil
}

func (r *Reader) Read() (*zed.Value, error) {
	r.once.Do(func() {
		go func() {
			r.err = run(r.runtime.Context(), r, r.runtime, nil)
			close(r.batchCh)
		}()
	})
next:
	if r.batch == nil {
		r.batch = <-r.batchCh
		if r.batch == nil {
			return nil, r.err
		}
	}
	vals := r.batch.Values()
	if r.index >= len(vals) {
		r.batch.Unref()
		r.batch, r.index = nil, 0
		goto next
	}
	rec := &vals[r.index]
	r.index++
	return rec, nil
}

func (r *Reader) Close() error {
	r.runtime.Context().Cancel()
	return r.Closer.Close()
}

func NewReader(ctx context.Context, program ast.Proc, zctx *zed.Context, reader zio.Reader) (*Reader, error) {
	pctx := proc.NewContext(ctx, zctx, nil)
	runtime, err := compiler.CompileForInternal(pctx, program, reader)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	r := &Reader{
		runtime: runtime,
		Closer:  io.NopCloser(nil),
		batchCh: make(chan zbuf.Batch),
	}
	if _, ok := reader.(io.Closer); ok {
		r.Closer = reader.(io.Closer)
	}
	return r, nil
}
