package zqd

import (
	"net/http"
	"sync/atomic"

	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqd/zeek"
	"go.uber.org/zap"
)

type Config struct {
	Root string
	// ZeekLauncher is the interface for launching zeek processes.
	ZeekLauncher zeek.Launcher
	Logger       *zap.Logger
}

type VersionMessage struct {
	Zqd string `json:"boomd"` //XXX boomd -> zqd
	Zq  string `json:"zq"`
}

// This struct filled in by main from linker setting version strings.
var Version VersionMessage

type Core struct {
	Root         string
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
	spaces, err := space.NewManager(conf.Root, conf.Logger)
	if err != nil {
		return nil, err
	}
	return &Core{
		Root:         conf.Root,
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
