package driver

import (
	"encoding/json"
	"fmt"
	"io"
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

// Global variable for driver package
// which determines whether this go process will
// (0) implement parallelism with local goroutines
// (1) implement parallelism by engaging multiple
// remote zqd /worker processes on the WorkerURLs list, or,
// (2) implement parallelism by calling a load-balanced
// Kubernetes service endpoint
const (
	PM_USE_GOROUTINES = iota
	PM_USE_WORKER_URLS
	PM_USE_SERVICE_ENDPOINT
)

// Default ParallelModel is local goroutines
var ParallelModel int = PM_USE_GOROUTINES
var WorkerServiceAddr string
var WorkerURLs []string

type parallelHead struct {
	pctx   *proc.Context
	parent proc.Interface
	once   sync.Once
	pg     *parallelGroup

	mu sync.Mutex // protects below
	sc ScannerCloser

	// workerConn is connection to a worker zqd process
	// that is only used for distributed zqd.
	// Thread (goroutine) parallelism is used when workerConn is nil.
	workerConn *api.Connection
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
			var sc ScannerCloser
			var err error
			if ph.workerConn == nil {
				// Thread (goroutine) parallelism uses nextSource
				sc, err = ph.pg.nextSource()

			} else {
				// Worker process parallelism uses nextSourceForConn
				sc, err = ph.pg.nextSourceForConn(ph.workerConn)
			}
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
	mcfg       MultiConfig
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

// nextSourceForConn is similar to nextSource, but instead of returning the scannerCloser
// for an open file (i.e. the stream for the open file),
// nextSourceForConn sends a request to a remote zqd worker process, and returns
// the ScannerCloser (i.e.output stream) for the remote zqd worker.
func (pg *parallelGroup) nextSourceForConn(conn *api.Connection) (ScannerCloser, error) {
	select {
	case src, ok := <-pg.sourceChan:
		if !ok {
			return nil, pg.sourceErr
		}

		req, err := pg.sourceToRequest(src)
		if err != nil {
			return nil, err
		}
		//jreq, _ := json.Marshal(*req)
		//println("Outbound request to worker: ", string(jreq))
		rc, err := conn.WorkerRaw(pg.pctx.Context, *req, nil) // rc is io.ReadCloser
		if err != nil {
			return nil, err
		}
		search := api.NewZngSearch(rc)
		s, err := scanner.NewScanner(pg.pctx.Context, search, nil, nil, req.Span)
		if err != nil {
			return nil, err
		}
		sc := struct {
			scanner.Scanner
			io.Closer
		}{s, rc}

		pg.mu.Lock()
		pg.scanners[sc] = struct{}{}
		pg.mu.Unlock()
		return sc, nil

	case <-pg.pctx.Done():
		return nil, pg.pctx.Err()
	}
}

func (pg *parallelGroup) doneSource(sc ScannerCloser) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.stats.Accumulate(sc.Stats())
	delete(pg.scanners, sc)
}

// sourceToRequest takes a Source and converts it into a WorkerRequest
func (pg *parallelGroup) sourceToRequest(src Source) (*api.WorkerRequest, error) {
	var req api.WorkerRequest
	if err := src.ToRequest(&req); err != nil {
		return nil, err
	}
	filterExpr := pg.filter.FilterExpr
	if filterExpr != nil {
		b, err := json.Marshal(filterToProc(filterExpr))
		if err != nil {
			return nil, err
		}
		req.Proc = b
	}
	req.Dir = pg.mcfg.Dir
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

func createParallelGroup(pctx *proc.Context, filt filter.Filter, filterExpr ast.BooleanExpr, msrc MultiSource, mcfg MultiConfig, workerURLs []string) ([]proc.Interface, *parallelGroup, error) {
	//println("createParallelGroup mcfg.Dir=", mcfg.Dir)
	pg := &parallelGroup{
		pctx: pctx,
		filter: SourceFilter{
			Filter:     filt,
			FilterExpr: filterExpr,
			Span:       mcfg.Span,
		},
		msrc:       msrc,
		mcfg:       mcfg,
		sourceChan: make(chan Source),
		scanners:   make(map[scanner.Scanner]struct{}),
	}

	var sources []proc.Interface
	// mcfg.UseWorkers is used to indicate that we are processing
	// a request that could be delegated to worker processes.
	// /search requests have mcfg.UseWorkers=true
	// If we are in a worker process, then mcfg.UseWorkers defaults to false.
	if !mcfg.UseWorkers || ParallelModel == PM_USE_GOROUTINES {
		sources = make([]proc.Interface, mcfg.Parallelism)
		for i := range sources {
			sources[i] = &parallelHead{pctx: pctx, pg: pg}
		}
	} else if ParallelModel == PM_USE_WORKER_URLS {
		// In this case each parallel head will be dedicated to a running zqd worker process
		for _, w := range workerURLs {
			sources = append(sources, &parallelHead{pctx: pctx, pg: pg, workerConn: api.NewConnectionTo(w)})
		}
	} else if ParallelModel == PM_USE_SERVICE_ENDPOINT {
		// In this case each parallel head will seperately request from a
		// load-balanced service endpoint (backed by an unspecified number of process instances)
		for i := 0; i < mcfg.Parallelism; i++ { // TODO: need to update mcfg.Parallelism in compile!
			sources = append(sources, &parallelHead{pctx: pctx, pg: pg,
				workerConn: api.NewConnectionTo(WorkerServiceAddr)})
		}
	} else {
		return sources, pg, fmt.Errorf("Unsupported ParallelModel %d", ParallelModel)
	}

	return sources, pg, nil
}
