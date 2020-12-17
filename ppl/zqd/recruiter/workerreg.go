package recruiter

import (
	"context"
	"flag"
	"net"
	"os"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"go.uber.org/zap"
)

type WorkerConfig struct {
	BoundWorkers string
	Fallback     bool
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
	fs.BoolVar(&c.Fallback, "worker.fallback", false, "fallback to using fewer workers than requested")
	fs.StringVar(&c.Host, "worker.host", "", "host ip of container")
	fs.IntVar(&c.IdleTime, "worker.idletime", 2000, "timeout in milliseconds for zombie worker processes to exit")
	fs.IntVar(&c.LongPoll, "worker.longpoll", 30000, "timeout in milliseconds for long poll of /recruiter/register request")
	fs.IntVar(&c.MaxRetry, "worker.maxretry", 10000, "maximum retry wait in milliseconds for registration request")
	fs.IntVar(&c.MinRetry, "worker.minretry", 200, "minimum retry wait in milliseconds for registration request")
	fs.StringVar(&c.Node, "worker.node", "", "logical node name within the compute cluster")
	fs.StringVar(&c.Recruiter, "worker.recruiter", "", "recruiter address for worker registration")
}

// WorkerReg maintains state for a workers's interactions
// with the recruiter and the zqd root process.
type WorkerReg struct {
	conf        WorkerConfig
	conn        *client.Connection
	logger      *zap.Logger
	releaseChan chan string
	selfaddr    string
}

func NewWorkerReg(srvAddr string, conf WorkerConfig, logger *zap.Logger) (*WorkerReg, error) {
	host, port, _ := net.SplitHostPort(srvAddr)
	if conf.Host != "" {
		host = conf.Host
	}
	w := &WorkerReg{
		conf:        conf,
		conn:        client.NewConnectionTo("http://" + conf.Recruiter),
		logger:      logger,
		releaseChan: make(chan string),
		selfaddr:    net.JoinHostPort(host, port),
	}
	return w, nil
}

// RegisterWithRecruiter is used by personality=worker.
func (w *WorkerReg) RegisterWithRecruiter() {
	ctx := context.Background()
	req := api.RegisterRequest{
		Timeout: w.conf.LongPoll,
		Worker: api.Worker{
			Addr:     w.selfaddr,
			NodeName: w.conf.Node,
		},
	}
	retryWait := w.conf.MinRetry
	// Loop for registration long polling.
	for {
		w.logger.Info("Register",
			zap.Int("longpoll", w.conf.LongPoll),
			zap.String("recruiter", w.conf.Recruiter))
		resp, err := w.conn.Register(ctx, req)
		if err != nil {
			w.logger.Error(
				"Error on recruiter registration, waiting to retry",
				zap.Int("retry", retryWait),
				zap.String("recruiter", w.conf.Recruiter),
				zap.Error(err))
			// Delay next request. There is an
			// exponential backoff on registration errors.
			time.Sleep(time.Duration(retryWait) * time.Millisecond)
			if retryWait < w.conf.MaxRetry {
				retryWait = (retryWait * 3) / 2
				// Note: doubling is too fast a backoff for this, so using 1.5 x.
			} else {
				retryWait = w.conf.MaxRetry
			}
			continue
		}
		retryWait = w.conf.MinRetry // Retry goes back to min after a success.
		if resp.Directive == "reserved" {
			w.logger.Info("Worker is reserved", zap.String("selfaddr", w.selfaddr))
			// Start listening to the releaseChannel.
			// Exit the nested loop on a release.
			// An idle timeout will cause the process to terminate.
			ticker := time.NewTicker(time.Duration(w.conf.IdleTime) * time.Millisecond)
			// GoDoc mentions fewer special cases with Ticker than with Timer.
			workerIsIdle := true
			// The worker "stays" in the ReservedLoop until it is finshed working for a given root process.
			// The worker starts in the idle state, and is only "busy" while responding to a /worker/chunksearch.
		ReservedLoop:
			for {
				select {
				case <-ticker.C:
					if workerIsIdle {
						w.logger.Warn("Worker timed out before receiving a request from the root",
							zap.String("selfaddr", w.selfaddr))
						os.Exit(0)
					}
				case msg := <-w.releaseChan:
					if msg == "release" {
						w.logger.Info("Worker is released", zap.String("selfaddr", w.selfaddr))
						// Breaking out of this nested loop will continue on to re-register.
						break ReservedLoop
					} else if msg == "idle" {
						workerIsIdle = true
						ticker = time.NewTicker(time.Duration(w.conf.IdleTime) * time.Millisecond)
						// Recreating the ticker is a safe way to reset it.
					} else { // Assume "busy".
						workerIsIdle = false
					}
				}
			}
		}
	}
}

// These three methods are called from the worker handlers.
// They do a nil check on the pointer receiver because if the
// -worker.bound flag is present then there may be no WorkerReg.
// The warnings on the default selector should not normally occur,
// and would indicate that something was broken.

func (w *WorkerReg) SendRelease() {
	if w != nil {
		select {
		case w.releaseChan <- "release":
		default:
			w.logger.Warn("Receiver not ready for release")
		}
	}
}

func (w *WorkerReg) SendBusy() {
	if w != nil {
		select {
		case w.releaseChan <- "busy":
		default:
			w.logger.Warn("Receiver not ready for busy")
		}
	}
}

func (w *WorkerReg) SendIdle() {
	if w != nil {
		select {
		case w.releaseChan <- "idle":
		default:
			w.logger.Warn("Receiver not ready for idle")
		}
	}
}
