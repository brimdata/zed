package zqd

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/pcapanalyzer"
	"github.com/brimsec/zq/ppl/zqd/space"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Config struct {
	Logger  *zap.Logger
	Root    string
	Version string

	Suricata pcapanalyzer.Launcher
	Zeek     pcapanalyzer.Launcher
}

type Core struct {
	logger    *zap.Logger
	registry  *prometheus.Registry
	root      iosrc.URI
	spaces    *space.Manager
	taskCount int64
	version   string

	suricata pcapanalyzer.Launcher
	zeek     pcapanalyzer.Launcher
}

func NewCore(ctx context.Context, conf Config) (*Core, error) {
	if conf.Logger == nil {
		conf.Logger = zap.NewNop()
	}
	root, err := iosrc.ParseURI(conf.Root)
	if err != nil {
		return nil, err
	}
	registry := prometheus.NewRegistry()
	spaces, err := space.NewManager(ctx, conf.Logger, registry, root)
	if err != nil {
		return nil, err
	}
	if conf.Version == "" {
		conf.Version = "unknown"
	}
	return &Core{
		logger:   conf.Logger,
		registry: registry,
		root:     root,
		spaces:   spaces,
		version:  conf.Version,
		suricata: conf.Suricata,
		zeek:     conf.Zeek,
	}, nil
}

func (c *Core) HasSuricata() bool {
	return c.suricata != nil
}

func (c *Core) HasZeek() bool {
	return c.zeek != nil
}

func (c *Core) Registry() *prometheus.Registry {
	return c.registry
}

func (c *Core) Root() iosrc.URI {
	return c.root
}

func (c *Core) nextTaskID() int64 {
	return atomic.AddInt64(&c.taskCount, 1)
}

func (c *Core) requestLogger(r *http.Request) *zap.Logger {
	return c.logger.With(zap.String("request_id", getRequestID(r.Context())))
}
