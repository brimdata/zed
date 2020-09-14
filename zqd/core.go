package zqd

import (
	"net/http"
	"sync/atomic"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqd/zeek"
	"go.uber.org/zap"
)

type Config struct {
	Root    string
	Version string
	// ZeekLauncher is the interface for launching zeek processes.
	ZeekLauncher zeek.Launcher
	Logger       *zap.Logger
}

type Core struct {
	Root         iosrc.URI
	Version      string
	ZeekLauncher zeek.Launcher
	spaces       *space.Manager
	taskCount    int64
	logger       *zap.Logger
}

func NewCore(conf Config) (*Core, error) {
	logger := conf.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	root, err := iosrc.ParseURI(conf.Root)
	if err != nil {
		return nil, err
	}
	spaces, err := space.NewManager(root, conf.Logger)
	if err != nil {
		return nil, err
	}
	version := conf.Version
	if version == "" {
		version = "unknown"
	}
	return &Core{
		Root:         root,
		Version:      version,
		ZeekLauncher: conf.ZeekLauncher,
		spaces:       spaces,
		logger:       logger,
	}, nil
}

func (c *Core) HasZeek() bool {
	return c.ZeekLauncher != nil
}

func (c *Core) requestLogger(r *http.Request) *zap.Logger {
	return c.logger.With(zap.String("request_id", getRequestID(r.Context())))
}

func (c *Core) getTaskID() int64 {
	return atomic.AddInt64(&c.taskCount, 1)
}
