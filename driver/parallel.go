package driver

import (
	"encoding/json"
	"sync"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zqd/api"
)

type parallelHead struct {
	pctx   *proc.Context
	parent proc.Interface
	once   sync.Once
	pg     *parallelGroup

	mu sync.Mutex // protects below
	sc ScannerCloser
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
	pctx       *proc.Context
	filter     SourceFilter
	msrc       MultiSource
	once       sync.Once
	sourceChan chan Source
	sourceErr  error

	mu       sync.Mutex // protects below
	stats    scanner.ScannerStats
	scanners map[scanner.Scanner]struct{}
}

func (pg *parallelGroup) nextSource() (ScannerCloser, error) {
	for {
		select {
		case src, ok := <-pg.sourceChan:
			if !ok {
				return nil, pg.sourceErr
			}
			sc, err := src.Open(pg.pctx.Context, pg.pctx.TypeContext, pg.filter)
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

func (pg *parallelGroup) doneSource(sc ScannerCloser) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.stats.Accumulate(sc.Stats())
	delete(pg.scanners, sc)
}

// XXX Note for mark: This function isn't yet used, but is an example of how
// to take a Source and fully convert it into a WorkerRequest:
//
// req, err := sourceToRequest(src, pg.filter.FilterExpr)
//
func sourceToRequest(src Source, filterExpr ast.BooleanExpr) (*api.WorkerRequest, error) {
	var req api.WorkerRequest
	if err := src.ToRequest(&req); err != nil {
		return nil, err
	}
	if filterExpr != nil {
		b, err := json.Marshal(filterToProc(filterExpr))
		if err != nil {
			return nil, err
		}
		req.Proc = b
	}
	return &req, nil
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
	pg.sourceErr = pg.msrc.SendSources(pg.pctx, pg.filter.Span, pg.sourceChan)
	close(pg.sourceChan)
}

func newCompareFn(fieldName string, reversed bool) (zbuf.RecordCmpFn, error) {
	if fieldName == "ts" {
		if reversed {
			return zbuf.CmpTimeReverse, nil
		} else {
			return zbuf.CmpTimeForward, nil
		}
	}
	fieldRead := ast.NewDotExpr(field.New(fieldName))
	res, err := expr.CompileExpr(fieldRead)
	if err != nil {
		return nil, err
	}
	rcmp := expr.NewCompareFn(true, res)
	return func(a, b *zng.Record) bool {
		return rcmp(a, b) < 0
	}, nil
}

func createParallelGroup(pctx *proc.Context, filt filter.Filter, filterExpr ast.BooleanExpr, msrc MultiSource, mcfg MultiConfig) ([]proc.Interface, *parallelGroup, error) {
	pg := &parallelGroup{
		pctx: pctx,
		filter: SourceFilter{
			Filter:     filt,
			FilterExpr: filterExpr,
			Span:       mcfg.Span,
		},
		msrc:       msrc,
		sourceChan: make(chan Source),
		scanners:   make(map[scanner.Scanner]struct{}),
	}

	sources := make([]proc.Interface, mcfg.Parallelism)
	for i := range sources {
		sources[i] = &parallelHead{pctx: pctx, parent: nil, pg: pg}
	}
	return sources, pg, nil
}
