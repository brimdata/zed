package worker

import (
	"context"
	"flag"
	"net"
	"os"
	"time"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/api/client"
	"go.uber.org/zap"
)

type WorkerConfig struct {
	BoundWorkers string
	Fallback     bool
	Host         string
	LongPoll     time.Duration
	MaxRetry     time.Duration
	MinRetry     time.Duration
	Node         string
	Recruiter    string
	IdleTime     time.Duration
}

func (c *WorkerConfig) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.BoundWorkers, "worker.bound", "", "bound workers as comma-separated [addr]:port list")
	fs.BoolVar(&c.Fallback, "worker.fallback", false, "fallback to using fewer workers than requested")
	fs.StringVar(&c.Host, "worker.host", "", "host ip of container")
	fs.DurationVar(&c.IdleTime, "worker.idletime", 10*time.Second, "timeout duration for zombie worker processes to exit")
	fs.DurationVar(&c.LongPoll, "worker.longpoll", 30*time.Second, "timeout duration for long poll of /recruiter/register request")
	fs.DurationVar(&c.MaxRetry, "worker.maxretry", 10*time.Second, "maximum retry wait duration for registration request")
	fs.DurationVar(&c.MinRetry, "worker.minretry", 200*time.Millisecond, "minimum retry wait duration for registration request")
	fs.StringVar(&c.Node, "worker.node", "", "logical node name within the compute cluster")
	fs.StringVar(&c.Recruiter, "worker.recruiter", "", "recruiter address for worker registration")
}

// RegistrationState maintains state for a workers's interactions
// with the recruiter and the zqd root process.
type RegistrationState struct {
	conf        WorkerConfig
	conn        *client.Connection
	ctx         context.Context
	logger      *zap.Logger
	releaseChan chan string
	selfaddr    string
}

func NewRegistrationState(ctx context.Context, srvAddr string, conf WorkerConfig, logger *zap.Logger) (*RegistrationState, error) {
	host, port, _ := net.SplitHostPort(srvAddr)
	if conf.Host != "" {
		host = conf.Host
	}
	rs := &RegistrationState{
		conf:        conf,
		conn:        client.NewConnectionTo("http://" + conf.Recruiter),
		ctx:         ctx,
		logger:      logger,
		releaseChan: make(chan string),
		selfaddr:    net.JoinHostPort(host, port),
	}
	return rs, nil
}

// RegisterWithRecruiter is used by personality=worker.
func (rs *RegistrationState) RegisterWithRecruiter() {
	req := api.RegisterRequest{
		Timeout: int(rs.conf.LongPoll / time.Millisecond),
		Worker: api.Worker{
			Addr:     rs.selfaddr,
			NodeName: rs.conf.Node,
		},
	}
	retryWait := rs.conf.MinRetry
	// Loop for registration long polling.
	for {
		rs.logger.Debug("Register",
			zap.Duration("longpoll", rs.conf.LongPoll),
			zap.String("recruiter", rs.conf.Recruiter))
		resp, err := rs.conn.Register(rs.ctx, req)
		if err != nil {
			rs.logger.Error(
				"Error on recruiter registration, waiting to retry",
				zap.Duration("retry", retryWait),
				zap.String("recruiter", rs.conf.Recruiter),
				zap.Error(err))
			// Delay next request. There is an
			// exponential backoff on registration errors.
			time.Sleep(retryWait)
			if retryWait < rs.conf.MaxRetry {
				retryWait = (retryWait * 3) / 2
				// Note: doubling is too fast a backoff for this, so using 1.5 x.
			} else {
				retryWait = rs.conf.MaxRetry
			}
			continue
		}
		retryWait = rs.conf.MinRetry // Retry goes back to min after a success.
		if resp.Directive != "reserved" {
			continue
		}
		rs.logger.Info("Worker is reserved", zap.String("selfaddr", rs.selfaddr))
		// Start listening to the releaseChannel.
		// Exit the nested loop on a release.
		// An idle timeout will cause the process to terminate.
		ticker := time.NewTicker(rs.conf.IdleTime)
		// GoDoc mentions fewer special cases with Ticker than with Timer.
		workerIsIdle := true
		// The worker "stays" in the ReservedLoop until it is finshed working for a given root process.
		// The worker starts in the idle state, and is only "busy" while responding to a /worker/chunksearch.
	ReservedLoop:
		for {
			select {
			case <-ticker.C:
				if workerIsIdle {
					rs.logger.Warn("Worker timed out before receiving a request from the root",
						zap.String("selfaddr", rs.selfaddr))
					os.Exit(0)
				}
			case msg := <-rs.releaseChan:
				if msg == "release" {
					rs.logger.Info("Worker is released", zap.String("selfaddr", rs.selfaddr))
					// Breaking out of this nested loop will continue on to re-register.
					break ReservedLoop
				} else if msg == "idle" {
					workerIsIdle = true
					ticker = time.NewTicker(rs.conf.IdleTime)
					// Recreating the ticker is a safe way to reset it.
				} else { // Assume "busy".
					workerIsIdle = false
				}
			}
		}
	}
}

// These three methods are called from the worker handlers.
// They do a nil check on the pointer receiver because if the
// -worker.bound flag is present then there may be no RegistrationState.
// The warnings on the default selector should not normally occur,
// and would indicate that something was broken.

func (rs *RegistrationState) SendRelease() {
	if rs != nil {
		select {
		case rs.releaseChan <- "release":
		default:
			rs.logger.Warn("Receiver not ready for release")
		}
	}
}

func (rs *RegistrationState) SendBusy() {
	if rs != nil {
		select {
		case rs.releaseChan <- "busy":
		default:
			rs.logger.Warn("Receiver not ready for busy")
		}
	}
}

func (rs *RegistrationState) SendIdle() {
	if rs != nil {
		select {
		case rs.releaseChan <- "idle":
		default:
			rs.logger.Warn("Receiver not ready for idle")
		}
	}
}
