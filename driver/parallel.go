package driver

import (
	"sync"

	"github.com/brimsec/zq/address"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type parallelHead struct {
	pctx   *proc.Context
	parent proc.Interface
	once   sync.Once
	pg     *parallelGroup

	mu sync.Mutex // protects below
	sc address.ScannerCloser
}

func (ph *parallelHead) closeOnDone() {
	<-ph.pctx.Done()
	ph.mu.Lock()
	if ph.sc != nil {
		ph.sc.Close()
		ph.sc = nil
	}
	ph.mu.Unlock()
}

func (ph *parallelHead) Pull() (zbuf.Batch, error) {
	// Trigger the parallel group to read from the multisource.
	ph.pg.once.Do(func() {
		go ph.pg.run()
	})
	// Ensure open scanners are closed when flowgraph execution stops.
	ph.once.Do(func() {
		go ph.closeOnDone()
	})

	ph.mu.Lock()
	defer ph.mu.Unlock()

	for {
		if ph.sc == nil {
			sc, err := ph.pg.nextSource()
			if sc == nil || err != nil {
				return nil, err
			}
			ph.sc = sc
		}
		batch, err := ph.sc.Pull()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			if err := ph.sc.Close(); err != nil {
				return nil, err
			}
			ph.pg.doneSource(ph.sc)
			ph.sc = nil
			continue
		}
		return batch, err
	}
}

func (ph *parallelHead) Done() {
	//XXX need to do something here... this happens when the scanner
	// hasn't finished but the flowgraph is done (e.g., tail).
	// I don't think this worked right prior to this refactor.
	// OR maybe this is ok because tail returns EOS then context is canceled?
}

type parallelGroup struct {
	pctx      *proc.Context
	filter    address.SourceFilter
	msrc      address.MultiSource
	once      sync.Once
	spanChan  chan address.SpanInfo
	sourceErr error

	mu       sync.Mutex // protects below
	stats    scanner.ScannerStats
	scanners map[scanner.Scanner]struct{}
}

//func (pg *parallelGroup) nextSource() (SpanInfo, error) {
// newSpanScanner needs to be around here instead of archive
func (pg *parallelGroup) nextSource() (address.ScannerCloser, error) {
	for {
		select {
		case spaninfo, ok := <-pg.spanChan:
			if !ok {
				return nil, pg.sourceErr
			}
			// here we will have a SpanInfo
			// local mode: copy code from closure in multisource here
			// distributed mode: recruit a worker here

			if len(pg.msrc.GetAltPaths()) == 0 {
				//archive.NewSpanScanner(pg.pctx, pg.msrc.ark, pg.pctx.TypeContext, pg.filter.Filter, pg.filter.FilterExpr, spaninfo)
			} else {
				// run code from closure in:
				//return m.chunkWalk(ctx, zctx, sf, srcChan)
			}

			sc, err := opener()
			if err != nil {
				return nil, err
			}
			if sc == nil {
				continue
			}
			pg.mu.Lock()
			pg.scanners[sc] = struct{}{}
			pg.mu.Unlock()
			return sc, nil
		case <-pg.pctx.Done():
			return nil, pg.pctx.Err()
		}
	}
}

func (pg *parallelGroup) doneSource(sc address.ScannerCloser) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.stats.Accumulate(sc.Stats())
	delete(pg.scanners, sc)
}

func (pg *parallelGroup) Stats() *scanner.ScannerStats {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	s := pg.stats
	for sc := range pg.scanners {
		s.Accumulate(sc.Stats())
	}
	return &s
}

func (pg *parallelGroup) run() {
	pg.sourceErr = pg.msrc.SendSources(pg.pctx, pg.pctx.TypeContext, pg.filter, pg.spanChan)
	close(pg.spanChan)
}

func newCompareFn(field string, reversed bool) (zbuf.RecordCmpFn, error) {
	if field == "ts" {
		if reversed {
			return zbuf.CmpTimeReverse, nil
		} else {
			return zbuf.CmpTimeForward, nil
		}
	}
	fieldRead := &ast.Field{
		Node:  ast.Node{Op: "Field"},
		Field: field,
	}
	res, err := expr.CompileExpr(fieldRead)
	if err != nil {
		return nil, err
	}
	rcmp := expr.NewCompareFn(true, res)
	return func(a, b *zng.Record) bool {
		return rcmp(a, b) < 0
	}, nil
}

func createParallelGroup(pctx *proc.Context, filterExpr ast.BooleanExpr, msrc address.MultiSource, mcfg address.MultiConfig) ([]proc.Interface, *parallelGroup, error) {
	var filt filter.Filter
	if filterExpr != nil {
		var err error
		if filt, err = filter.Compile(filterExpr); err != nil {
			return nil, nil, err
		}
	}
	pg := &parallelGroup{
		pctx: pctx,
		filter: address.SourceFilter{
			Filter:     filt,
			FilterExpr: filterExpr,
			Span:       mcfg.Span,
		},
		msrc:     msrc,
		spanChan: make(chan address.SpanInfo),
		scanners: make(map[scanner.Scanner]struct{}),
	}

	sources := make([]proc.Interface, mcfg.Parallelism)
	for i := range sources {
		sources[i] = &parallelHead{pctx: pctx, parent: nil, pg: pg}
	}
	// here we could return a group of recruited workers, which must be Proc's
	// so create a Proc which is a worker interface, needs Pull())
	return sources, pg, nil
}
