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
	"github.com/brimdata/zed/lake/index"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Update(ctx context.Context, lk lakeapi.Interface, conf Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	indexes, err := lakeapi.GetIndexRules(ctx, lk)
	if err != nil {
		return err
	}
	branches, err := getBranches(ctx, conf, indexes, lk, logger)
	if err != nil {
		return err
	}
	group, ctx := errgroup.WithContext(ctx)
	for _, branch := range branches {
		branch := branch
		branch.logger.Info("updating pool", zap.Object("config", branch))
		group.Go(func() error {
			head, err := branch.head(ctx)
			if err != nil {
				return err
			}
			for _, task := range branch.tasks {
				if _, err := task.run(ctx, head); err != nil {
					task.logger().Error("task error", zap.Error(err))
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
	indexes, err := lakeapi.GetIndexRules(ctx, lk)
	if err != nil {
		return err
	}
	branches, err := getBranches(ctx, conf, indexes, lk, logger)
	if err != nil {
		return err
	}
	if len(branches) == 0 {
		logger.Info("no pools found")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	monitors := make(map[ksuid.KSUID]*monitor)
	for _, pool := range branches {
		monitorBranch(ctx, pool, monitors)
	}
	return listen(ctx, monitors, conf, indexes, conn, logger)
}

func getBranches(ctx context.Context, conf Config, indexes []index.Rule, lk lakeapi.Interface, logger *zap.Logger) ([]*branch, error) {
	pools, err := lakeapi.GetPools(ctx, lk)
	if err != nil {
		return nil, err
	}
	var branches []*branch
	for _, pool := range pools {
		b, err := newBranch(conf, pool, indexes, lk, logger)
		if err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	return branches, nil
}

func listen(ctx context.Context, monitors map[ksuid.KSUID]*monitor, conf Config,
	indexes []index.Rule, conn *client.Connection, logger *zap.Logger) error {
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
			detail := detail.(*api.EventPool)
			pool, err := lakeapi.LookupPoolByID(ctx, lk, detail.PoolID)
			if err != nil {
				return err
			}
			b, err := newBranch(conf, pool, indexes, lk, logger)
			if err != nil {
				return err
			}
			monitorBranch(ctx, b, monitors)
		case "pool-delete":
			detail := detail.(*api.EventPool)
			if m, ok := monitors[detail.PoolID]; ok {
				m.cancel()
				delete(monitors, detail.PoolID)
				m.branch.logger.Info("pool deleted")
			}
		case "branch-commit":
			detail := detail.(*api.EventBranchCommit)
			if m, ok := monitors[detail.PoolID]; ok && m.branch.name == detail.Branch {
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
		b.logger.Info("monitoring pool", zap.Object("config", b))
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
		threads = append(threads, newThread(ctx, b, t))
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
	branch  *branch
	task    branchTask
	running int32
}

func newThread(ctx context.Context, branch *branch, task branchTask) *thread {
	return &thread{ctx: ctx, branch: branch, task: task}
}

func (t *thread) run() {
	if !atomic.CompareAndSwapInt32(&t.running, 0, 1) {
		return
	}
	t.task.logger().Info("thread running")
	go func() {
		defer atomic.StoreInt32(&t.running, 0)
		timer := time.NewTimer(0)
		<-timer.C
		var head ksuid.KSUID
		for t.ctx.Err() == nil {
			current, err := t.branch.head(t.ctx)
			if err != nil {
				t.task.logger().Error("error fetching branch head", zap.Error(err))
				return
			}
			if current == head {
				t.task.logger().Info("thread exiting")
				return
			}
			head = current
			next, err := t.task.run(t.ctx, head)
			if err != nil {
				t.task.logger().Error("thread exited with error", zap.Error(err))
				return
			}
			if next == nil {
				// This means there's no further work, but before exiting check
				// to see if there are any new commits since the task was run.
				continue
			}
			sleep := time.Until(*next)
			t.task.logger().Debug("sleeping", zap.Duration("duration", sleep))
			timer.Reset(sleep)
			select {
			case <-timer.C:
			case <-t.ctx.Done():
			}
		}
	}()
}
