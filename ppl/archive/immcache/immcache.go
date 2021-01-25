// Package immcache contains facilities for caching immutable files for an
// archive.
package immcache

import (
	"context"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/prometheus/client_golang/prometheus"
)

type ImmutableCache interface {
	ReadFile(context.Context, iosrc.URI) ([]byte, error)
}

type Config struct {
	// LocalCacheSize specifies the number of immutable files to keep in a
	// local lru cache used to speed up searches. Values less than or equal to 0
	// (default), disables local caching of immutable files.
	LocalCacheSize int
}

func New(conf Config, reg prometheus.Registerer) (ImmutableCache, error) {
	if conf.LocalCacheSize > 0 {
		return NewLocalCache(conf.LocalCacheSize, reg)
	}
	return nil, nil
}
