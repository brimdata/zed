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
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Update(ctx context.Context, lk lakeapi.Interface, conf Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	if err := conf.Index.getRules(ctx, lk); err != nil {
		return err
	}
	branches, err := getBranches(ctx, conf, lk, logger)
	if err != nil {
		return err
	}
	group, ctx := errgroup.WithContext(ctx)
	for _, branch := range branches {
		branch := branch
		group.Go(func() error {
			for _, task := range branch.tasks {
				if _, err := task(ctx); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return group.Wait()
}

func Monitor(ctx context.Context, conn *client.Connection, conf Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	for {
		switch err := runMonitor(ctx, conf, conn, logger); {
		case errors.Is(err, syscall.ECONNREFUSED):
			logger.Info("cannot connect to lake, retrying in 5 seconds")
		case err != nil:
			return err
		}
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func runMonitor(ctx context.Context, conf Config, conn *client.Connection, logger *zap.Logger) error {
	lk := lakeapi.NewRemoteLake(conn)
	if err := conf.Index.getRules(ctx, lk); err != nil {
		return err
	}
	branches, err := getBranches(ctx, conf, lk, logger)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	monitors := make(map[ksuid.KSUID]*monitor)
	for _, pool := range branches {
		monitorBranch(ctx, pool, monitors)
	}
	return listen(ctx, monitors, conf, conn, logger)
}

func getBranches(ctx context.Context, conf Config, lk lakeapi.Interface, logger *zap.Logger) ([]*branch, error) {
	pools, err := lakeapi.GetPools(ctx, lk)
	if err != nil {
		return nil, err
	}
	var branches []*branch
	for _, pool := range pools {
		// XXX For now only track main but this should be configurable and allow
		// non-main branches as well as monitoring multiple branches.
		branches = append(branches, newBranch("main", pool, lk, conf, logger))
	}
	return branches, nil
}

func listen(ctx context.Context, monitors map[ksuid.KSUID]*monitor, conf Config, conn *client.Connection, logger *zap.Logger) error {
	ev, err := conn.SubscribeEvents(ctx)
	if err != nil {
		return err
	}
	defer ev.Close()
	lk := lakeapi.NewRemoteLake(conn)
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
			pool, err := lakeapi.LookupPoolByID(ctx, lk, ev.PoolID)
			if err != nil {
				return err
			}
			b := newBranch("main", pool, lk, conf, logger)
			monitorBranch(ctx, b, monitors)
		case "pool-delete":
			ev := detail.(*api.EventPool)
			if m, ok := monitors[ev.PoolID]; ok {
				m.cancel()
				delete(monitors, ev.PoolID)
			}
		case "branch-commit":
			ev := detail.(*api.EventBranchCommit)
			if m, ok := monitors[ev.PoolID]; ok && m.branch.name == ev.Branch {
				m.run()
			}
		case "branch-update", "branch-delete":
			// Ignore these events.
		default:
			logger.Warn("unexpected event kind received", zap.String("kind", kind))
		}
	}
}

func monitorBranch(ctx context.Context, b *branch, monitors map[ksuid.KSUID]*monitor) {
	if _, ok := monitors[b.pool.ID]; !ok {
		m := newMonitor(ctx, b)
		monitors[b.pool.ID] = m
		m.run()
	}
}

type monitor struct {
	branch  *branch
	cancel  context.CancelFunc
	threads []*thread
}

func newMonitor(ctx context.Context, b *branch) *monitor {
	ctx, cancel := context.WithCancel(ctx)
	var threads []*thread
	for _, t := range b.tasks {
		threads = append(threads, newThread(ctx, t))
	}
	return &monitor{branch: b, cancel: cancel, threads: threads}
}

func (b *monitor) run() {
	for _, t := range b.threads {
		t.run()
	}
}

type thread struct {
	ctx     context.Context
	exec    func(context.Context) (*time.Time, error)
	running int32
}

func newThread(ctx context.Context, exec func(context.Context) (*time.Time, error)) *thread {
	return &thread{ctx: ctx, exec: exec}
}

func (t *thread) run() {
	if !atomic.CompareAndSwapInt32(&t.running, 0, 1) {
		return
	}
	go func() {
		timer := time.NewTimer(0)
		<-timer.C
		for t.ctx.Err() == nil {
			next, err := t.exec(t.ctx)
			if err != nil || next == nil {
				return
			}
			timer.Reset(time.Until(*next))
			select {
			case <-timer.C:
			case <-t.ctx.Done():
			}
		}
	}()
}
