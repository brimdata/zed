package space

import (
	"context"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ppl/zqd/storage/archivestore"
	"go.uber.org/zap"
)

type compactor struct {
	cancel  context.CancelFunc
	done    chan struct{}
	logger  *zap.Logger
	manager *Manager
	notify  chan api.SpaceID
}

func newCompactor(manager *Manager) *compactor {
	ctx, cancel := context.WithCancel(context.Background())
	c := &compactor{
		cancel:  cancel,
		done:    make(chan struct{}),
		logger:  manager.logger.Named("compactor"),
		manager: manager,
		notify:  make(chan api.SpaceID, 5),
	}
	go c.run(ctx)
	return c
}

func (c *compactor) WriteComplete(id api.SpaceID) {
	c.notify <- id
}

func (c *compactor) run(ctx context.Context) {
	for id := range c.notify {
		c.compact(ctx, id)
	}
	close(c.done)
}

func (c *compactor) compact(ctx context.Context, id api.SpaceID) {
	logger := c.logger.With(zap.String("space_id", string(id)))
	sp, err := c.manager.Get(id)
	if err != nil {
		logger.Warn("Space does not exist")
		return
	}
	store, ok := sp.Storage().(*archivestore.Storage)
	if !ok {
		return
	}
	ctx, cancel, err := sp.StartOp(ctx)
	if err != nil {
		logger.Info("Could not aquire space op", zap.Error(err))
		return
	}
	defer cancel()

	logger.Info("Compaction started")
	start := time.Now()
	if err := store.Compact(ctx); err != nil {
		if err == context.Canceled {
			logger.Info("Compaction aborted")
		} else {
			logger.Warn("Compaction error", zap.Error(err))
		}
		return
	}
	logger.Info("compaction completed", zap.Duration("duration", time.Since(start)))
}

func (c *compactor) close() {
	close(c.notify)
	c.cancel()
	<-c.done
}
