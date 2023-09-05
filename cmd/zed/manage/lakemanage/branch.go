package lakemanage

import (
	"context"

	"github.com/brimdata/zed/api"
	lakeapi "github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"go.uber.org/zap"
)

type branch struct {
	config PoolConfig
	lake   lakeapi.Interface
	logger *zap.Logger
	pool   *pools.Config
	tasks  []branchTask
}

func newBranch(c Config, pool *pools.Config, lake lakeapi.Interface, logger *zap.Logger) *branch {
	config := c.poolConfig(pool)
	logger = logger.Named("pool").With(
		zap.String("name", pool.Name),
		zap.Stringer("id", pool.ID),
		zap.String("branch", config.Branch),
		zap.Duration("interval", config.interval()),
	)
	if config.interval() == 0 {
		logger.Info("Manage disabled for branch")
		return nil
	}
	b := &branch{
		config: config,
		lake:   lake,
		logger: logger,
		pool:   pool,
	}
	b.tasks = append(b.tasks, &compactTask{branch: b})
	return b
}

type branchTask interface {
	run(context.Context) error
}

type compactTask struct {
	*branch
}

func (c *compactTask) run(ctx context.Context) error {
	c.logger.Debug("compaction started")
	head := lakeparse.Commitish{Pool: c.pool.Name, Branch: c.config.Branch}
	it, err := NewPoolDataObjectIterator(ctx, c.lake, &head, c.pool.SortKey)
	if err != nil {
		return err
	}
	defer it.Close()
	ch := make(chan Run)
	go func() {
		err = CompactionScan(ctx, it, c.pool, ch)
		close(ch)
	}()
	var found int
	var compacted int
	for run := range ch {
		commit, err := c.lake.Compact(ctx, c.pool.ID, c.config.Branch, run.ObjectIDs(), false, api.CommitMessage{})
		if err != nil {
			return err
		}
		found++
		compacted += len(run.Objects)
		c.logger.Debug("compacted", zap.Stringer("commit", commit), zap.Int("objects_compacted", len(run.Objects)))
	}
	level := zap.InfoLevel
	if compacted == 0 {
		level = zap.DebugLevel
	}
	c.logger.Log(level, "compaction completed", zap.Int("runs_found", found), zap.Int("objects_compacted", compacted))
	return err
}
