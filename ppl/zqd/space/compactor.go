package space

import (
	"context"
	"sync"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ppl/zqd/storage/archivestore"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type compactor struct {
	cancel      context.CancelFunc
	done        sync.WaitGroup
	logger      *zap.Logger
	manager     *Manager
	notify      chan api.SpaceID
	compactDone chan api.SpaceID
	sem         *semaphore.Weighted
}

const maxConcurrentCompacts = 1

func newCompactor(manager *Manager) *compactor {
	ctx, cancel := context.WithCancel(context.Background())
	c := &compactor{
		cancel:      cancel,
		logger:      manager.logger.Named("compactor"),
		manager:     manager,
		notify:      make(chan api.SpaceID),
		compactDone: make(chan api.SpaceID),
		sem:         semaphore.NewWeighted(maxConcurrentCompacts),
	}
	go c.run(ctx)
	return c
}

func (c *compactor) WriteComplete(id api.SpaceID) {
	c.notify <- id
}

func (c *compactor) launchCompact(ctx context.Context, id api.SpaceID) {
	go func() {
		if err := c.sem.Acquire(ctx, 1); err != nil {
			return
		}
		defer c.sem.Release(1)
		defer func() { c.compactDone <- id }()
		c.compact(ctx, id)
	}()
}

func (c *compactor) run(ctx context.Context) {
	c.done.Add(1)
	active := make(map[api.SpaceID]bool)
	for {
		select {
		case id := <-c.notify:
			_, ok := active[id]
			if !ok {
				// No compaction active for this space right now.
				active[id] = false
				c.launchCompact(ctx, id)
				continue
			}
			// When the current compaction is done, start another.
			active[id] = true
		case id := <-c.compactDone:
			again := active[id]
			if again {
				active[id] = false
				c.launchCompact(ctx, id)
				continue
			}
			delete(active, id)
		case <-ctx.Done():
			c.done.Done()
			return
		}
	}
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
	logger.Info("Compaction completed", zap.Duration("duration", time.Since(start)))
}

func (c *compactor) close() {
	close(c.notify)
	c.cancel()
	c.done.Wait()
}
