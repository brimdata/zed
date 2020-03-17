package zqd

import (
	"net/http"
	"sync/atomic"

	"go.uber.org/zap"
)

type Config struct {
	Root string
	// The exact path of the zeek executable. If this is an empty string zeek
	// will be located from $PATH. This is needed in the
	// POST /space/:space/packet endpoint.
	ZeekExec string
	// SortLimit specifies the limit of logs in posted pcap to sort. Its
	// existence is only as a hook for testing.  Eventually zqd will sort an
	// unlimited amount of logs and this can be taken out.
	SortLimit int
	Logger    *zap.Logger
}

type VersionMessage struct {
	Zqd string `json:"boomd"` //XXX boomd -> zqd
	Zq  string `json:"zq"`
}

// This struct filled in by main from linker setting version strings.
var Version VersionMessage

type Core struct {
	Root string
	// The exact path of the zeek executable. If this is an empty string zeek
	// will be located from $PATH. This is needed in the
	// POST /space/:space/packet endpoint.
	ZeekExec string
	// SortLimit specifies the limit of logs in posted pcap to sort. Its
	// existence is only as a hook for testing.  Eventually zqd will sort an
	// unlimited amount of logs and this can be taken out.
	SortLimit int
	taskCount int64
	logger    *zap.Logger
}

func NewCore(conf Config) *Core {
	logger := conf.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Core{
		Root:      conf.Root,
		ZeekExec:  conf.ZeekExec,
		SortLimit: conf.SortLimit,
		logger:    logger,
	}
}

func (c *Core) requestLogger(r *http.Request) *zap.Logger {
	return c.logger.With(zap.String("request_id", getRequestID(r.Context())))
}

func (c *Core) getTaskID() int64 {
	return atomic.AddInt64(&c.taskCount, 1)
}
