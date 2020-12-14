package recruiter

import (
	"context"
	"flag"
	"net"
	"sync"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"go.uber.org/zap"
)

type WorkerConfig struct {
	// BoundWorkers is a fixed list of workers bound to a root process.
	// It is used for ZTests and simple clusters without a recruiter.
	BoundWorkers string
	Host         string
	LongPoll     int
	MaxRetry     int
	MinRetry     int
	Node         string
	Recruiter    string
	Retry        int
	IdleTime     int
}

func (c *WorkerConfig) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.BoundWorkers, "worker.bound", "", "bound workers as comma-separated [addr]:port list")
	fs.StringVar(&c.Host, "worker.host", "", "host ip of container")
	fs.IntVar(&c.LongPoll, "worker.longpoll", 30000, "timeout in milliseconds for long poll of /recruiter/register request")
	fs.IntVar(&c.MaxRetry, "worker.maxretry", 10000, "maximum retry wait in milliseconds for registration request")
	fs.IntVar(&c.MinRetry, "worker.minretry", 200, "minimum retry wait in milliseconds for registration request")
	fs.StringVar(&c.Node, "worker.node", "", "logical node name within the compute cluster")
	fs.StringVar(&c.Recruiter, "worker.recruiter", "", "recruiter address for worker registration")
	fs.IntVar(&c.IdleTime, "worker.idletime", 2000, "timeout in milliseconds for zombie worker processes to exit")
}

type WorkerReg struct {
	conf           WorkerConfig
	conn           *client.Connection
	logger         *zap.Logger
	releaseChan    chan bool
	selfaddr       string
	SearchLock     sync.Mutex
	zombieDuration time.Duration
	zombieTimer    *time.Timer
}

func NewWorkerReg(srvAddr string, conf WorkerConfig, logger *zap.Logger) (*WorkerReg, error) {
	host, port, _ := net.SplitHostPort(srvAddr)
	if conf.Host != "" {
		host = conf.Host
	}
	w := &WorkerReg{
		conf:           conf,
		conn:           client.NewConnectionTo("http://" + conf.Recruiter),
		logger:         logger,
		releaseChan:    make(chan bool),
		selfaddr:       net.JoinHostPort(host, port),
		zombieDuration: time.Duration(conf.IdleTime) * time.Millisecond,
	}

	return w, nil
}

func (w *WorkerReg) RegisterWithRecruiter() {
	ctx := context.Background()

	registerreq := api.RegisterRequest{
		Timeout: w.conf.LongPoll,
		Worker: api.Worker{
			Addr:     w.selfaddr,
			NodeName: w.conf.Node,
		},
	}

	// Loop while long polling.
	// Exponential backoff on registration errors.
	retryWait := w.conf.MinRetry
	for {
		w.logger.Info("Register",
			zap.Int("longpoll", w.conf.LongPoll),
			zap.String("recruiter", w.conf.Recruiter))

		resp, err := w.conn.Register(ctx, registerreq)
		if err != nil {
			w.logger.Error(
				"Error on recruiter registration, waiting to retry",
				zap.Int("retry", retryWait),
				zap.String("recruiter", w.conf.Recruiter),
				zap.Error(err))
			time.Sleep(time.Duration(retryWait) * time.Millisecond)
			if retryWait < w.conf.MaxRetry {
				retryWait = (retryWait * 3) / 2
				// Note: doubling seems too fast a backoff for this, so using 1.5 x
			} else {
				retryWait = w.conf.MaxRetry
			}
			continue
		}
		retryWait = w.conf.MinRetry

		if resp.Directive == "reserved" {
			w.logger.Info("worker is reserved", zap.String("selfaddr", w.selfaddr))
			select {
			case <-w.releaseChan:
				w.logger.Info("worker is released", zap.String("selfaddr", w.selfaddr))
			}
		}
	}
}

func (w *WorkerReg) Release() {
	w.releaseChan <- true
}
