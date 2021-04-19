package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"go.uber.org/zap"
)

type Driver interface {
	Warn(msg string) error
	Write(channelID int, batch zbuf.Batch) error
	ChannelEnd(channelID int) error
	Stats(api.ScannerStats) error
}

type Config struct {
	Custom    kernel.Hook
	Logger    *zap.Logger
	Span      nano.Span
	StatsTick <-chan time.Time
}

func RunWithReader(ctx context.Context, d Driver, program ast.Proc, zctx *zson.Context, reader zio.Reader, cfg Config) error {
	pctx := proc.NewContext(ctx, zctx, cfg.Logger)
	runtime, err := compiler.CompileForInternal(pctx, program, reader, cfg.Custom)
	if err != nil {
		pctx.Cancel()
		return err
	}
	return run(pctx, d, runtime, nil)
}

func RunWithOrderedReader(ctx context.Context, d Driver, program ast.Proc, zctx *zson.Context, reader zio.Reader, layout order.Layout, logger *zap.Logger) error {
	pctx := proc.NewContext(ctx, zctx, logger)
	runtime, err := compiler.CompileForInternalWithOrder(pctx, program, reader, layout)
	if err != nil {
		pctx.Cancel()
		return err
	}
	return run(pctx, d, runtime, nil)
}

func RunWithFileSystem(ctx context.Context, d Driver, program ast.Proc, zctx *zson.Context, reader zio.Reader, adaptor proc.DataAdaptor) (zbuf.ScannerStats, error) {
	pctx := proc.NewContext(ctx, zctx, nil)
	runtime, err := compiler.CompileForFileSystem(pctx, program, reader, adaptor)
	if err != nil {
		pctx.Cancel()
		return zbuf.ScannerStats{}, err
	}
	err = run(pctx, d, runtime, nil)
	return runtime.Statser().Stats(), err
}

func RunJoinWithFileSystem(ctx context.Context, d Driver, program ast.Proc, zctx *zson.Context, readers []zio.Reader, adaptor proc.DataAdaptor) (zbuf.ScannerStats, error) {
	pctx := proc.NewContext(ctx, zctx, nil)
	runtime, err := compiler.CompileJoinForFileSystem(pctx, program, readers, adaptor)
	if err != nil {
		pctx.Cancel()
		return zbuf.ScannerStats{}, err
	}
	err = run(pctx, d, runtime, nil)
	return runtime.Statser().Stats(), err
}

func RunWithLake(ctx context.Context, d Driver, program ast.Proc, zctx *zson.Context, lake proc.DataAdaptor) (zbuf.ScannerStats, error) {
	pctx := proc.NewContext(ctx, zctx, nil)
	runtime, err := compiler.CompileForLake(pctx, program, lake, 0)
	if err != nil {
		pctx.Cancel()
		return zbuf.ScannerStats{}, err
	}
	err = run(pctx, d, runtime, nil)
	return runtime.Statser().Stats(), err
}

func RunWithLakeAndStats(ctx context.Context, d Driver, program ast.Proc, zctx *zson.Context, lake proc.DataAdaptor, ticker <-chan time.Time, logger *zap.Logger, parallelism int) error {
	pctx := proc.NewContext(ctx, zctx, logger)
	runtime, err := compiler.CompileForLake(pctx, program, lake, parallelism)
	if err != nil {
		pctx.Cancel()
		return err
	}
	return run(pctx, d, runtime, ticker)
}

func run(pctx *proc.Context, d Driver, runtime *compiler.Runtime, statsTicker <-chan time.Time) error {
	puller := runtime.AsPuller()
	if puller == nil {
		pctx.Cancel()
		return errors.New("internal error: driver called with unprepared runtime")
	}
	statser := runtime.Statser()
	if statser == nil && statsTicker != nil {
		pctx.Cancel()
		return errors.New("internal error: driver wants live stats but runtime has no statser")
	}
	pullerCh := make(chan zbuf.Batch)
	done := make(chan error)
	defer func() {
		pctx.Cancel()
		<-pullerCh
		<-done
	}()
	go func() {
		for {
			// We can simply call Pull here knowing it will return
			// when pctx.Cancel() is called on exit from our
			// parent goroutine and we won't block because pullerCh
			// and done are always read on exit.
			batch, err := safePull(puller)
			if batch == nil || err != nil {
				close(pullerCh)
				if err != nil {
					done <- err
				}
				close(done)
				return
			}
			pullerCh <- batch
		}
	}()
	for {
		select {
		case <-statsTicker:
			if err := d.Stats(api.ScannerStats(statser.Stats())); err != nil {
				return err
			}
		case batch := <-pullerCh:
			if batch == nil {
				err := <-done
				if endErr := d.ChannelEnd(0); err == nil {
					err = endErr
				}
				if err == nil && statser != nil {
					err = d.Stats(api.ScannerStats(statser.Stats()))
				}
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
				return err
			}
			// We will get rid of channel ID... client SearchReults
			// protocol currently uses it.  See issue #2652.
			if err := d.Write(0, batch); err != nil {
				return err
			}
		case warning := <-pctx.Warnings:
			if err := d.Warn(warning); err != nil {
				return err
			}
		}
	}
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
	w zio.Writer
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
func Copy(ctx context.Context, w zio.Writer, prog ast.Proc, zctx *zson.Context, r zio.Reader, cfg Config) error {
	d := &transformDriver{w: w}
	return RunWithReader(ctx, d, prog, zctx, r, cfg)
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

func (*Reader) Warn(warning string) error          { return nil }
func (*Reader) Stats(stats api.ScannerStats) error { return nil }
func (*Reader) ChannelEnd(cid int) error           { return nil }

func (r *Reader) Write(_ int, batch zbuf.Batch) error {
	if batch != nil {
		r.batchCh <- batch
	}
	return nil
}

func (r *Reader) Read() (*zng.Record, error) {
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
	if r.index >= r.batch.Length() {
		r.batch.Unref()
		r.batch, r.index = nil, 0
		goto next
	}
	rec := r.batch.Index(r.index)
	r.index++
	return rec, nil
}

func (r *Reader) Close() error {
	r.runtime.Context().Cancel()
	return r.Closer.Close()
}

func NewReader(ctx context.Context, program ast.Proc, zctx *zson.Context, reader zio.Reader) (*Reader, error) {
	return NewReaderWithConfig(ctx, Config{}, program, zctx, reader)
}

func NewReaderWithConfig(ctx context.Context, conf Config, program ast.Proc, zctx *zson.Context, reader zio.Reader) (*Reader, error) {
	pctx := proc.NewContext(ctx, zctx, conf.Logger)
	runtime, err := compiler.CompileForInternal(pctx, program, reader, conf.Custom)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	r := &Reader{
		runtime: runtime,
		Closer:  ioutil.NopCloser(nil),
		batchCh: make(chan zbuf.Batch),
	}
	if _, ok := reader.(io.Closer); ok {
		r.Closer = reader.(io.Closer)
	}
	return r, nil
}
