package driver

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/brimdata/zq/compiler"
	"github.com/brimdata/zq/compiler/ast"
	"github.com/brimdata/zq/compiler/kernel"
	"github.com/brimdata/zq/field"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/ppl/zqd/worker"
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng/resolver"
	"go.uber.org/zap"
)

// XXX ReaderSortKey should be a field.Static.  Issue #1467.
type Config struct {
	Custom            kernel.Hook
	Logger            *zap.Logger
	ReaderSortKey     string
	ReaderSortReverse bool
	Span              nano.Span
	StatsTick         <-chan time.Time
}

type scannerProc struct {
	zbuf.Scanner
}

func (s *scannerProc) Done() {}

type namedScanner struct {
	zbuf.Scanner
	name string
}

func (n *namedScanner) Pull() (zbuf.Batch, error) {
	b, err := n.Scanner.Pull()
	if err != nil {
		err = fmt.Errorf("%s: %w", n.name, err)
	}
	return b, err
}

func compile(ctx context.Context, program ast.Proc, zctx *resolver.Context, readers []zbuf.Reader, cfg Config) (*muxOutput, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.Span.Dur == 0 {
		cfg.Span = nano.MaxSpan
	}
	var sortKey field.Static
	if cfg.ReaderSortKey != "" {
		sortKey = field.New(cfg.ReaderSortKey)
	}
	runtime, err := compiler.NewWithSortedInput(zctx, program, sortKey, cfg.ReaderSortReverse)
	if err != nil {
		return nil, err
	}
	if err := runtime.Optimize(); err != nil {
		return nil, err
	}
	procs := make([]proc.Interface, 0, len(readers))
	scanners := make([]zbuf.Scanner, 0, len(readers))
	for _, r := range readers {
		sn, err := zbuf.NewScanner(ctx, r, runtime, cfg.Span)
		if err != nil {
			return nil, err
		}
		if stringer, ok := r.(fmt.Stringer); ok {
			sn = &namedScanner{sn, stringer.String()}
		}
		scanners = append(scanners, sn)
		procs = append(procs, &scannerProc{sn})
	}
	pctx := &proc.Context{
		Context:  ctx,
		Logger:   cfg.Logger,
		Warnings: make(chan string, 5),
		Zctx:     zctx,
	}
	if err := runtime.Compile(cfg.Custom, pctx, procs); err != nil {
		return nil, err
	}
	return newMuxOutput(pctx, runtime.Outputs(), zbuf.MultiStats(scanners)), nil
}

type MultiConfig struct {
	Custom      kernel.Hook
	Distributed bool // true if remote request specified worker count
	Order       zbuf.Order
	Logger      *zap.Logger
	Parallelism int
	Span        nano.Span
	StatsTick   <-chan time.Time
	Worker      worker.WorkerConfig
}

func compileMulti(ctx context.Context, program ast.Proc, zctx *resolver.Context, msrc MultiSource, mcfg MultiConfig) (*muxOutput, error) {
	if mcfg.Logger == nil {
		mcfg.Logger = zap.NewNop()
	}
	if mcfg.Span.Dur == 0 {
		mcfg.Span = nano.MaxSpan
	}
	if mcfg.Parallelism == 0 {
		mcfg.Parallelism = runtime.GOMAXPROCS(0)
	}

	sortKey, sortReversed := msrc.OrderInfo()
	runtime, err := compiler.NewWithSortedInput(zctx, program, sortKey, sortReversed)
	if err != nil {
		return nil, err
	}
	if err := runtime.Optimize(); err != nil {
		return nil, err
	}
	if !runtime.IsParallelizable() {
		mcfg.Parallelism = 1
	}
	pctx := &proc.Context{
		Context:  ctx,
		Logger:   mcfg.Logger,
		Warnings: make(chan string, 5),
		Zctx:     zctx,
	}
	sources, pgroup, err := createParallelGroup(pctx, runtime, msrc, mcfg)
	if err != nil {
		return nil, err
	}
	if len(sources) > 1 {
		runtime.Parallelize(len(sources))
	}
	if err := runtime.Compile(mcfg.Custom, pctx, sources); err != nil {
		return nil, err
	}
	return newMuxOutput(pctx, runtime.Outputs(), pgroup), nil
}
