package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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
