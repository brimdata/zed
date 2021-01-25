package archive

import (
	"context"
	"path"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/chunk"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type smallFileGetter interface {
	ReadFile(context.Context, iosrc.URI) ([]byte, error)
}

type smallFileCache struct {
	getter smallFileGetter
	lru    *lru.ARCCache
	hits   *prometheus.CounterVec
	misses *prometheus.CounterVec
}

func newSmallFileCache(size int, getter smallFileGetter, registerer prometheus.Registerer) (*smallFileCache, error) {
	lru, err := lru.NewARC(size)
	if err != nil {
		return nil, err
	}
	if registerer == nil {
		registerer = prometheus.NewRegistry()
	}
	factory := promauto.With(registerer)
	return &smallFileCache{
		getter: getter,
		lru:    lru,
		hits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "archive_cache_hits_total",
				Help: "Number of hits for a cache lookup.",
			},
			[]string{"kind"},
		),
		misses: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "archive_cache_misses_total",
				Help: "Number of misses for a cache lookup.",
			},
			[]string{"kind"},
		),
	}, nil
}

func (c *smallFileCache) ReadFile(ctx context.Context, u iosrc.URI) ([]byte, error) {
	kind, _, _ := chunk.FileMatch(path.Base(u.Path))
	if v, ok := c.lru.Get(u.String()); ok {
		c.hits.WithLabelValues(kind.Description()).Inc()
		return v.([]byte), nil
	}
	b, err := c.getter.ReadFile(ctx, u)
	if err != nil {
		return nil, err
	}
	c.lru.Add(u.String(), b)
	c.misses.WithLabelValues(kind.Description()).Inc()
	return b, nil
}
