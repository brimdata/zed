package driver

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/ppl/zqd/recruiter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
)

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
	workerConn *client.Connection
	label      string
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
				sc, err = ph.pg.nextSourceForConn(ph.workerConn, ph.label)
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
func (pg *parallelGroup) nextSourceForConn(conn *client.Connection, label string) (ScannerCloser, error) {
	select {
	case src, ok := <-pg.sourceChan:
		if !ok {
			return nil, pg.sourceErr
		}

		req, err := pg.sourceToRequest(src)
		if err != nil {
			return nil, err
		}

		//println(time.Now().Unix()%10000, "request to", label, "chunks", req.ChunkPaths[0])
		rc, err := conn.WorkerRaw(pg.pctx.Context, *req, nil) // rc is io.ReadCloser
		if err != nil {
			return nil, err
		}
		search := client.NewZngSearch(rc)
		s, err := zbuf.NewScanner(pg.pctx.Context, search, nil, req.Span)
		if err != nil {
			return nil, err
		}
		sc := struct {
			zbuf.Scanner
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
	if filterExpr := pg.filter.FilterExpr; filterExpr != nil {
		b, err := json.Marshal(filterToProc(filterExpr))
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

func createParallelGroup(pctx *proc.Context, filterExpr ast.BooleanExpr, msrc MultiSource, mcfg MultiConfig) ([]proc.Interface, *parallelGroup, error) {
	pg := &parallelGroup{
		pctx: pctx,
		filter: SourceFilter{
			FilterExpr: filterExpr,
			Span:       mcfg.Span,
		},
		msrc:       msrc,
		mcfg:       mcfg,
		sourceChan: make(chan Source),
		scanners:   make(map[zbuf.Scanner]struct{}),
	}

	var sources []proc.Interface
	var err error
	if mcfg.DistributedExec {
		workers, err := recruiter.RecruitWorkers(pctx, mcfg.Parallelism)
		if err != nil {
			return nil, nil, err
		}
		var connectionsToRelease []*client.Connection
		for _, w := range workers {
			conn := client.NewConnectionTo("http://" + w)
			connectionsToRelease = append(connectionsToRelease, conn)
			sources = append(sources, &parallelHead{
				pctx: pctx, pg: pg, workerConn: conn, label: w})
		}
		go pg.releaseWorkersOnDone(connectionsToRelease)
	} else {
		// This is the code path used by the zqd daemon for Brim.
		for i := 0; i < mcfg.Parallelism; i++ {
			sources = append(sources, &parallelHead{pctx: pctx, pg: pg})
		}
	}
	return sources, pg, err
}

func (pg *parallelGroup) releaseWorkersOnDone(conns []*client.Connection) {
	<-pg.pctx.Done()
	for _, conn := range conns {
		recruiter.ReleaseWorker(pg.pctx, conn)
	}
}
