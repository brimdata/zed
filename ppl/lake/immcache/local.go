package immcache

import (
	"context"
	"path"

	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/ppl/lake/chunk"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
)

type LocalCache struct {
	metrics
	lru *lru.ARCCache
}

func NewLocalCache(size int, registerer prometheus.Registerer) (*LocalCache, error) {
	lru, err := lru.NewARC(size)
	if err != nil {
		return nil, err
	}
	if registerer == nil {
		registerer = prometheus.NewRegistry()
	}
	return &LocalCache{
		metrics: newMetrics(registerer),
		lru:     lru,
	}, nil
}

func (c *LocalCache) ReadFile(ctx context.Context, u iosrc.URI) ([]byte, error) {
	kind, _, _ := chunk.FileMatch(path.Base(u.Path))
	if v, ok := c.lru.Get(u.String()); ok {
		c.hits.WithLabelValues(kind.Description()).Inc()
		return v.([]byte), nil
	}
	b, err := iosrc.ReadFile(ctx, u)
	if err != nil {
		return nil, err
	}
	c.lru.Add(u.String(), b)
	c.misses.WithLabelValues(kind.Description()).Inc()
	return b, nil
}
