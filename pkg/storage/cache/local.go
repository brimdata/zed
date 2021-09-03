package cache

import (
	"context"
	"path"

	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/pkg/storage"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
)

type LocalCache struct {
	storage.Engine
	metrics
	lru       *lru.ARCCache
	cacheable Cacheable
}

func NewLocalCache(engine storage.Engine, cacheable Cacheable, size int, registerer prometheus.Registerer) (*LocalCache, error) {
	lru, err := lru.NewARC(size)
	if err != nil {
		return nil, err
	}
	if registerer == nil {
		registerer = prometheus.NewRegistry()
	}
	return &LocalCache{
		Engine:    engine,
		metrics:   newMetrics(registerer),
		cacheable: cacheable,
		lru:       lru,
	}, nil
}

func (c *LocalCache) Get(ctx context.Context, u *storage.URI) (storage.Reader, error) {
	if !c.cacheable(u) {
		return c.Engine.Get(ctx, u)
	}
	kind, _, _ := data.FileMatch(path.Base(u.Path))
	if v, ok := c.lru.Get(u.String()); ok {
		c.hits.WithLabelValues(kind.Description()).Inc()
		return storage.NewBytesReader(v.([]byte)), nil
	}
	b, err := c.Engine.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	c.lru.Add(u.String(), b)
	c.misses.WithLabelValues(kind.Description()).Inc()
	return b, nil
}
