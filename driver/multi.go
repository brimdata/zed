package driver

import (
	"context"
	"fmt"
	"io"

	"github.com/brimsec/zq/address"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
)

type oneSource struct {
	r            zbuf.Reader
	sortKey      string
	sortReversed bool
}

func (o *oneSource) GetAltPaths() []string { return []string{} }

func (o *oneSource) OrderInfo() (field string, reversed bool) {
	return o.sortKey, o.sortReversed
}

func (o *oneSource) SendSources(ctx context.Context, _ *resolver.Context, sf address.SourceFilter, c chan address.SpanInfo) error {
	scanner, err := scanner.NewScanner(ctx, o.r, sf.Filter, sf.FilterExpr, sf.Span)
	if err != nil {
		return err
	}
	if stringer, ok := o.r.(fmt.Stringer); ok {
		scanner = &namedScanner{scanner, stringer.String()}
	}
	f := func() (address.ScannerCloser, error) {
		return &scannerCloser{
			Scanner: scanner,
			Closer:  &onClose{},
		}, nil
	}
	select {
	case c <- f:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func rdrToMulti(r zbuf.Reader, cfg Config) (address.MultiSource, address.MultiConfig) {
	msrc := &oneSource{
		r:            r,
		sortKey:      cfg.ReaderSortKey,
		sortReversed: cfg.ReaderSortReverse,
	}
	mcfg := address.MultiConfig{
		Custom:      cfg.Custom,
		Logger:      cfg.Logger,
		Parallelism: 1,
		Span:        cfg.Span,
		StatsTick:   cfg.StatsTick,
		Warnings:    cfg.Warnings,
	}
	return nil, mcfg
}

func zbufDirInt(reversed bool) int {
	if reversed {
		return -1
	}
	return 1
}

type scannerCloser struct {
	scanner.Scanner
	io.Closer
}

type onClose struct {
	fn func() error
}

func (c *onClose) Close() error {
	if c.fn == nil {
		return nil
	}
	return c.fn()
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
