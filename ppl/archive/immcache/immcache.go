// Package immcache contains facilities for caching immutable files for an
// archive.
package immcache

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Kind string

const (
	KindNone  Kind = "none"
	KindLocal Kind = "local"
	KindRedis Kind = "redis"
)

func (k *Kind) Set(s string) error {
	switch s {
	case "none", "":
		*k = KindNone
	case "local":
		*k = KindLocal
	case "redis":
		*k = KindRedis
	default:
		return fmt.Errorf("unknown immutable cache kind: %q", s)
	}
	return nil
}

func (k Kind) String() string {
	return string(k)
}

type ImmutableCache interface {
	ReadFile(context.Context, iosrc.URI) ([]byte, error)
}

type Config struct {
	Kind Kind
	// LocalCacheSize specifies the number of immutable files to keep in a
	// local lru cache used to speed up searches. Values less than or equal to 0
	// (default), disables local caching of immutable files.
	LocalCacheSize int
	// RedisKeyExpiration is the time redis will keep the key of the immutable
	// file around before deleting this. If RedisKeyExpiration == 0 the key will
	// be kept around indefinitely. Because this is supposed to be used as an
	// LRU cache, if the redis server eviction policy is set volitale-lru the
	// value of RedisKeyExpiration should not be set to 0.
	RedisKeyExpiration time.Duration
}

func (c *Config) SetFlags(fs *flag.FlagSet) {
	fs.Var(&c.Kind, "immcache.kind", "kind of immutable cache")
	fs.IntVar(&c.LocalCacheSize, "immcache.local.size", 128, "number of small files to keep in local cache")
	fs.DurationVar(&c.RedisKeyExpiration, "immcache.redis.keyexpiry", time.Hour*24, "expiration duration of immutable keys")
}

func New(conf Config, rclient *redis.Client, reg prometheus.Registerer) (ImmutableCache, error) {
	switch conf.Kind {
	case KindLocal:
		return NewLocalCache(conf.LocalCacheSize, reg)
	case KindRedis:
		return NewRedisCache(rclient, conf, reg), nil
	}
	return nil, nil
}

type metrics struct {
	hits   *prometheus.CounterVec
	misses *prometheus.CounterVec
}

func newMetrics(reg prometheus.Registerer) metrics {
	factory := promauto.With(reg)
	return metrics{
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
	}
}
