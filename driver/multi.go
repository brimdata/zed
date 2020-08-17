package driver

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/zap"
)

// A MultiSource is a set of one or more ZNG record sources, which could be
// a zar archive, a local directory, a collection of remote objects, etc.
type MultiSource interface {
	// OrderInfo reports if the same order exists on the data in any
	// single source and on the sources reported via SendSources.
	OrderInfo() (field string, reversed bool)

	// SendSources sends SourceOpeners to the given channel. If the
	// MultiSource declares an ordering via its OrderInfo method, the
	// SourceOpeners must be sent in the same order.
	// The MultiSource should return a nil error when all of its sources
	// have been sent; it must not close the provided channel.
	// SendSources is called from a single goroutine, but the SourceOpeners
	// it generates will be used from potentially many goroutines. For best
	// performance, SendSources should perform quick filtering that performs
	// little or no i/o, and let the returned ScannerCloser perform more intensive
	// filtering (e.g., reading a micro-index to check for filter matching).
	SendSources(context.Context, *resolver.Context, SourceFilter, chan SourceOpener) error
}

// A SourceOpener is a closure sent by a MultiSource to provide scanning
// access to a single source. It may return a nil ScannerCloser, in the
// case that it represents a logically empty source.
type SourceOpener func() (ScannerCloser, error)

type ScannerCloser interface {
	scanner.Scanner
	io.Closer
}

type SourceFilter struct {
	Filter     filter.Filter
	FilterExpr ast.BooleanExpr
	Span       nano.Span
}

type MultiConfig struct {
	Custom      proc.Compiler
	Logger      *zap.Logger
	Parallelism int
	Span        nano.Span
	StatsTick   <-chan time.Time
	Warnings    chan string
}

type oneSource struct {
	r            zbuf.Reader
	sortKey      string
	sortReversed bool
}

func (o *oneSource) OrderInfo() (field string, reversed bool) {
	return o.sortKey, o.sortReversed
}

func (o *oneSource) SendSources(ctx context.Context, _ *resolver.Context, sf SourceFilter, c chan SourceOpener) error {
	scanner, err := scanner.NewScanner(ctx, o.r, sf.Filter, sf.FilterExpr, sf.Span)
	if err != nil {
		return err
	}
	if stringer, ok := o.r.(fmt.Stringer); ok {
		scanner = &namedScanner{scanner, stringer.String()}
	}
	f := func() (ScannerCloser, error) {
		return &noCloseScanner{scanner}, nil
	}
	select {
	case c <- f:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func rdrToMulti(r zbuf.Reader, cfg Config) (MultiSource, MultiConfig) {
	msrc := &oneSource{
		r:            r,
		sortKey:      cfg.ReaderSortKey,
		sortReversed: cfg.ReaderSortReverse,
	}
	mcfg := MultiConfig{
		Custom:      cfg.Custom,
		Logger:      cfg.Logger,
		Parallelism: 1,
		Span:        cfg.Span,
		StatsTick:   cfg.StatsTick,
		Warnings:    cfg.Warnings,
	}
	return msrc, mcfg
}

func zbufDirInt(reversed bool) int {
	if reversed {
		return -1
	}
	return 1
}

type noCloseScanner struct {
	scanner.Scanner
}

func (r *noCloseScanner) Close() error {
	return nil
}

type namedScanner struct {
	scanner.Scanner
	name string
}

func (n *namedScanner) Pull() (zbuf.Batch, error) {
	b, err := n.Scanner.Pull()
	if err != nil {
		err = fmt.Errorf("%s: %w", n.name, err)
	}
	return b, err
}
