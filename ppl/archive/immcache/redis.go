package immcache

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
)

type RedisCache struct {
	metrics
	expiry time.Duration
	client *redis.Client
}

func NewRedisCache(client *redis.Client, conf Config, reg prometheus.Registerer) *RedisCache {
	return &RedisCache{
		metrics: newMetrics(reg),
		expiry:  time.Duration(conf.RedisKeyExpiration),
		client:  client,
	}
}

func (c *RedisCache) ReadFile(ctx context.Context, u iosrc.URI) ([]byte, error) {
	kind, _, _ := chunk.FileMatch(path.Base(u.Path))
	res := c.client.Get(ctx, u.String())
	if err := res.Err(); err == nil {
		c.hits.WithLabelValues(kind.Description()).Inc()
		return res.Bytes()
	} else if !errors.Is(err, redis.Nil) {
		return nil, err
	}

	b, err := iosrc.ReadFile(ctx, u)
	if err != nil {
		return nil, err
	}

	c.misses.WithLabelValues(kind.Description()).Inc()
	return b, c.client.Set(ctx, u.String(), b, c.expiry).Err()
}
