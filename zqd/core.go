package zqd

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/brimsec/zq/zqd/zeek"
	"go.uber.org/zap"
)

type Config struct {
	Root string
	// ZeekLauncher is the interface for launching zeek processes.
	ZeekLauncher zeek.Launcher
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
	Root         string
	ZeekLauncher zeek.Launcher
	// SortLimit specifies the limit of logs in posted pcap to sort. Its
	// existence is only as a hook for testing.  Eventually zqd will sort an
	// unlimited amount of logs and this can be taken out.
	SortLimit int
	taskCount int64
	logger    *zap.Logger

	// ingestLock protects the ingests map and the deletePending
	// field inside the ingestWaitState's.
	ingestLock sync.Mutex
	ingests    map[string]*ingestWaitState
}

type ingestWaitState struct {
	deletePending int
	// WaitGroup for active ingests
	wg sync.WaitGroup
	// closed to signal active ingest should terminate
	cancelChan chan struct{}
}

func NewCore(conf Config) *Core {
	logger := conf.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Core{
		Root:         conf.Root,
		ZeekLauncher: conf.ZeekLauncher,
		SortLimit:    conf.SortLimit,
		logger:       logger,
		ingests:      make(map[string]*ingestWaitState),
	}
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

func (c *Core) startSpaceIngest(ctx context.Context, space string) (context.Context, func(), bool) {
	c.ingestLock.Lock()
	defer c.ingestLock.Unlock()

	iws, ok := c.ingests[space]
	if !ok {
		iws = &ingestWaitState{
			cancelChan: make(chan struct{}, 0),
		}
		c.ingests[space] = iws
	}
	if iws.deletePending > 0 {
		return ctx, func() {}, false
	}

	ctx, cancel := context.WithCancel(ctx)
	iws.wg.Add(1)
	ingestDone := func() {
		iws.wg.Done()
		cancel()
	}

	go func() {
		select {
		case <-ctx.Done():
		case <-iws.cancelChan:
			cancel()
		}
	}()

	return ctx, ingestDone, true
}

func (c *Core) startSpaceDelete(space string) func() {
	c.ingestLock.Lock()

	iws, ok := c.ingests[space]
	if !ok {
		iws = &ingestWaitState{
			cancelChan: make(chan struct{}, 0),
		}
		c.ingests[space] = iws
	}
	if iws.deletePending == 0 {
		close(iws.cancelChan)
	}
	iws.deletePending++

	c.ingestLock.Unlock()

	iws.wg.Wait()

	return func() {
		c.ingestLock.Lock()
		defer c.ingestLock.Unlock()
		iws.deletePending--
		if iws.deletePending == 0 {
			delete(c.ingests, space)
		}
	}
}
