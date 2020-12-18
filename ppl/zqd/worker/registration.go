package worker

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

// RegistrationState maintains state for a workers's interactions
// with the recruiter and the zqd root process.
type RegistrationState struct {
	conf        WorkerConfig
	conn        *client.Connection
	logger      *zap.Logger
	releaseChan chan string
	selfaddr    string
}

func NewRegistrationState(srvAddr string, conf WorkerConfig, logger *zap.Logger) (*RegistrationState, error) {
	host, port, _ := net.SplitHostPort(srvAddr)
	if conf.Host != "" {
		host = conf.Host
	}
	rs := &RegistrationState{
		conf:        conf,
		conn:        client.NewConnectionTo("http://" + conf.Recruiter),
		logger:      logger,
		releaseChan: make(chan string),
		selfaddr:    net.JoinHostPort(host, port),
	}
	return rs, nil
}

// RegisterWithRecruiter is used by personality=worker.
func (rs *RegistrationState) RegisterWithRecruiter() {
	ctx := context.Background()
	req := api.RegisterRequest{
		Timeout: rs.conf.LongPoll,
		Worker: api.Worker{
			Addr:     rs.selfaddr,
			NodeName: rs.conf.Node,
		},
	}
	retryWait := rs.conf.MinRetry
	// Loop for registration long polling.
	for {
		rs.logger.Info("Register",
			zap.Int("longpoll", rs.conf.LongPoll),
			zap.String("recruiter", rs.conf.Recruiter))
		resp, err := rs.conn.Register(ctx, req)
		if err != nil {
			rs.logger.Error(
				"Error on recruiter registration, waiting to retry",
				zap.Int("retry", retryWait),
				zap.String("recruiter", rs.conf.Recruiter),
				zap.Error(err))
			// Delay next request. There is an
			// exponential backoff on registration errors.
			time.Sleep(time.Duration(retryWait) * time.Millisecond)
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
		ticker := time.NewTicker(time.Duration(rs.conf.IdleTime) * time.Millisecond)
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
					ticker = time.NewTicker(time.Duration(rs.conf.IdleTime) * time.Millisecond)
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
