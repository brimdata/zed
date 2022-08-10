package lakemanage

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	lakeapi "github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

type monitor struct {
	conn   *client.Connection
	config Config
	lake   lakeapi.Interface
	logger *zap.Logger
}

func Monitor(ctx context.Context, conn *client.Connection, config Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	m := &monitor{
		conn:   conn,
		config: config,
		lake:   lakeapi.NewRemoteLake(conn),
		logger: logger,
	}
	timer := time.NewTimer(0)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()
	for {
		switch err := m.run(ctx); {
		case errors.Is(err, syscall.ECONNREFUSED):
			m.logger.Info("cannot connect to lake, retrying in 5 seconds")
		case err != nil:
			return err
		}
		timer.Reset(time.Second * 5)
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *monitor) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pools, err := lakeapi.GetPools(ctx, m.lake)
	if err != nil {
		return err
	}
	monitors := make(map[ksuid.KSUID]*poolMonitor)
	for _, pool := range pools {
		m.launchPool(ctx, pool, monitors)
	}
	return m.listen(ctx, monitors)
}

func (m *monitor) listen(ctx context.Context, monitors map[ksuid.KSUID]*poolMonitor) error {
	ev, err := m.conn.SubscribeEvents(ctx)
	if err != nil {
		return err
	}
	defer ev.Close()
	for {
		kind, detail, err := ev.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				// Ignore EOF error from lost connection.
				return nil
			}
			return err
		}
		switch kind {
		case "pool-new":
			ev := detail.(*api.EventPool)
			pool, err := lakeapi.LookupPoolByID(ctx, m.lake, ev.PoolID)
			if err != nil {
				return err
			}
			m.launchPool(ctx, pool, monitors)
		case "pool-delete":
			ev := detail.(*api.EventPool)
			if pm, ok := monitors[ev.PoolID]; ok {
				pm.cancel()
				delete(monitors, ev.PoolID)
			}
		case "branch-commit":
			ev := detail.(*api.EventBranchCommit)
			if pm, ok := monitors[ev.PoolID]; ok && pm.branch == ev.Branch {
				pm.run()
			}
		case "branch-update", "branch-delete":
			// Ignore these events.
		default:
			m.logger.Warn("unexpected event kind received", zap.String("kind", kind))
		}
	}
}

func (m *monitor) launchPool(ctx context.Context, pool *pools.Config, monitors map[ksuid.KSUID]*poolMonitor) {
	if _, ok := monitors[pool.ID]; !ok {
		// XXX For now only track main but this should be configurable and allow
		// non-main branches as well as monitoring multiple branches.
		pm := newPoolMonitor(ctx, pool, "main", m.lake, m.config, m.logger)
		pm.run()
		monitors[pool.ID] = pm
	}
}

type poolMonitor struct {
	branch  string
	cancel  context.CancelFunc
	config  Config
	ctx     context.Context
	logger  *zap.Logger
	lake    lakeapi.Interface
	pool    *pools.Config
	running int32
}

func newPoolMonitor(ctx context.Context, pool *pools.Config, branch string, lk lakeapi.Interface, config Config, logger *zap.Logger) *poolMonitor {
	ctx, cancel := context.WithCancel(ctx)
	return &poolMonitor{
		branch: branch,
		cancel: cancel,
		config: config,
		ctx:    ctx,
		lake:   lk,
		pool:   pool,
		logger: logger.Named("pool").With(
			zap.String("name", pool.Name),
			zap.Stringer("id", pool.ID),
			zap.String("branch", branch),
		),
	}
}

func (p *poolMonitor) run() {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return
	}
	go func() {
		timer := time.NewTimer(0)
		defer func() {
			if !timer.Stop() {
				<-timer.C
			}
			p.logger.Debug("exit")
		}()
		for p.ctx.Err() == nil {
			next, err := p.scan()
			if err != nil || next == nil {
				if err != nil && !errors.Is(err, context.Canceled) {
					p.logger.Error("scan error", zap.Error(err))
				}
				atomic.StoreInt32(&p.running, 0)
				return
			}
			timer.Reset(time.Until(*next))
			select {
			case <-p.ctx.Done():
			case <-timer.C:
			}
		}
	}()
}

func (p *poolMonitor) scan() (*time.Time, error) {
	p.logger.Debug("scan started")
	head := lakeparse.Commitish{Pool: p.pool.Name, Branch: p.branch}
	reader, err := NewPoolObjectIterator(p.ctx, p.lake, &head, p.pool.Layout)
	if err != nil {
		return nil, err
	}
	var nextcold *time.Time
	ch := make(chan Run)
	go func() {
		nextcold, err = Scan(p.ctx, reader, p.pool, p.config.ColdThreshold, ch)
		close(ch)
	}()
	var found int
	var compacted int
	for run := range ch {
		found++
		compacted += len(run.Objects)
		commit, err := p.lake.Compact(p.ctx, p.pool.ID, head.Branch, run.ObjectIDs(), api.CommitMessage{})
		if err != nil {
			return nil, err
		}
		p.logger.Debug("compacted", zap.Stringer("commit", commit), zap.Int("objects_compacted", len(run.Objects)))
	}
	if compacted == 0 {
		p.logger.Debug("scan completed", zap.Int("runs_found", found), zap.Int("objects_compacted", compacted))
	} else {
		p.logger.Info("scan completed", zap.Int("runs_found", found), zap.Int("objects_compacted", compacted))
	}
	return nextcold, err
}
