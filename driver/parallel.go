package driver

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/compiler"
	"github.com/brimsec/zq/ppl/zqd/recruiter"
	"github.com/brimsec/zq/ppl/zqd/worker"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"go.uber.org/zap"
)

type namedScannerCloser struct {
	zbuf.Scanner
	io.Closer
	name string
}

func (nsc namedScannerCloser) String() string {
	return nsc.name
}

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
			println("xyzzy", "parallelHead.Pull", err.Error(), "sc.name", ph.sc.String(),
				"BytesRead", ph.sc.Stats().BytesRead, "RecordsRead", ph.sc.Stats().RecordsRead)
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
	filter     SourceFilter
	logger     *zap.Logger
	mcfg       MultiConfig
	msrc       MultiSource
	once       sync.Once
	pctx       *proc.Context
	sourceErr  error
	sourceChan chan Source

	mu       sync.Mutex // protects below
	stats    zbuf.ScannerStats
	scanners map[zbuf.Scanner]struct{}
}

func (pg *parallelGroup) nextSource() (ScannerCloser, error) {
	for {
		select {
		case src, ok := <-pg.sourceChan:
			if !ok {
				return nil, pg.sourceErr
			}
			var sc ScannerCloser
			var err error
			if pg.mcfg.Distributed {
				sc, err = pg.getRemoteScannerCloser(pg.pctx, src, pg.mcfg.Worker, pg.mcfg.Logger)
			} else {
				sc, err = src.Open(pg.pctx.Context, pg.pctx.TypeContext, pg.filter)
			}
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

func (pg *parallelGroup) getRemoteScannerCloser(pctx *proc.Context, src Source, conf worker.WorkerConfig, logger *zap.Logger) (ScannerCloser, error) {
	conn, err := recruiter.GetWorkerConnection(pctx, conf, logger)
	if err != nil {
		return nil, err
	}
	req, err := pg.sourceToRequest(src)
	if err != nil {
		return nil, err
	}
	rc, err := conn.WorkerChunkSearch(pg.pctx.Context, *req, nil) // rc is io.ReadCloser
	if err != nil {
		return nil, err
	}
	search := client.NewZngSearch(rc)
	s, err := zbuf.NewScanner(pg.pctx.Context, search, nil, req.Span)
	if err != nil {
		return nil, err
	}
	return namedScannerCloser{s, rc, conn.ClientHostURL()}, nil
}

func (pg *parallelGroup) doneSource(sc ScannerCloser) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.stats.Accumulate(sc.Stats())
	delete(pg.scanners, sc)
}

// sourceToRequest takes a Source and converts it into a WorkerChunkRequest.
func (pg *parallelGroup) sourceToRequest(src Source) (*api.WorkerChunkRequest, error) {
	var req api.WorkerChunkRequest
	if err := src.ToRequest(&req); err != nil {
		return nil, err
	}
	if filterExpr := pg.filter.Filter; filterExpr != nil {
		b, err := json.Marshal(filterExpr.AsProc())
		if err != nil {
			return nil, err
		}
		req.Proc = b
	}
	req.Dir = pg.mcfg.Order.Int()
	return &req, nil
}

func (pg *parallelGroup) Stats() *zbuf.ScannerStats {
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

func createParallelGroup(pctx *proc.Context, filter *compiler.Filter, msrc MultiSource, mcfg MultiConfig) ([]proc.Interface, *parallelGroup, error) {
	pg := &parallelGroup{
		filter: SourceFilter{
			Filter: filter,
			Span:   mcfg.Span,
		},
		logger:     mcfg.Logger,
		mcfg:       mcfg,
		msrc:       msrc,
		pctx:       pctx,
		scanners:   make(map[zbuf.Scanner]struct{}),
		sourceChan: make(chan Source),
	}
	var sources []proc.Interface
	for i := 0; i < mcfg.Parallelism; i++ {
		sources = append(sources, &parallelHead{pctx: pctx, pg: pg})
	}
	return sources, pg, nil
}
