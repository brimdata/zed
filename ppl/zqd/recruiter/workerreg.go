package recruiter

import (
	"context"
	"flag"
	"net"
	"os"
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
	MaxRetry     int
	MinRetry     int
	Node         string
	Recruiter    string
	Retry        int
	Timeout      int
	ZTimeout     int
}

func (c *WorkerConfig) SetWorkerFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.BoundWorkers, "worker.bound", "", "bound workers as comma-separated [addr]:port list")
	fs.StringVar(&c.Host, "worker.host", "", "host ip of container")
	fs.IntVar(&c.Timeout, "worker.maxretry", 10000, "maximum retry wait in milliseconds for registration request")
	fs.IntVar(&c.Timeout, "worker.minretry", 200, "minimum retry wait in milliseconds for registration request")
	fs.StringVar(&c.Node, "worker.node", "", "logical node name within the compute cluster")
	fs.StringVar(&c.Recruiter, "worker.recruiter", "", "recruiter address for worker registration")
	fs.IntVar(&c.Timeout, "worker.timeout", 30000, "timeout in milliseconds for long poll of /recruiter/register request")
	fs.IntVar(&c.ZTimeout, "worker.ztimeout", 2000, "timeout in milliseconds for zombie worker processes to exit")
}

type WorkerReg struct {
	conf           *WorkerConfig
	conn           *client.Connection
	ctx            context.Context // context from Run() function
	logger         *zap.Logger
	selfaddr       string
	SearchLock     sync.Mutex
	zombieDuration time.Duration
	zombieTimer    *time.Timer
}

func NewWorkerReg(ctx context.Context, srvAddr string, logger *zap.Logger, conf *WorkerConfig) (*WorkerReg, error) {

	host, port, _ := net.SplitHostPort(srvAddr)
	if conf.Host != "" {
		host = conf.Host
	}
	w := &WorkerReg{
		conf:           conf,
		conn:           client.NewConnectionTo("http://" + conf.Recruiter),
		ctx:            ctx,
		logger:         logger,
		selfaddr:       net.JoinHostPort(host, port),
		zombieDuration: time.Duration(conf.ZTimeout) * time.Millisecond,
	}

	// Create a timer to exit this process if it is not receiving work,
	// but leave it in the stopped state. Note that the channel must be cleared.
	w.zombieTimer = time.NewTimer(w.zombieDuration)
	if !w.zombieTimer.Stop() {
		<-w.zombieTimer.C
	}
	go w.zombieKiller() // this goroutine listens for a zombieTimout and exits

	return w, nil
}

// RegisterWithRecruiter is supposed to be run as a goroutine,
// so context for requests will come from Run() of the listen command.
func (w *WorkerReg) RegisterWithRecruiter() {
	println("here I am...")
	// Workers in the process of registration cannot be zombies.
	w.StopZombieTimer()
	defer w.StartZombieTimer()
	println("here I am again...")

	// This should be a loop that tries to reregister, called as a goroutine.
	// Loop should be suspended when a /worker/search is in progress, and
	// resume afterwards.
	// So, break out of loop when reserved, then register is called again on /worker/release
	// Failure case is when /worker/release is not called. Maybe we need some locks and timers
	// to take care of that.
	registerreq := api.RegisterRequest{
		Timeout: w.conf.Timeout,
		Worker: api.Worker{
			Addr:     w.selfaddr,
			NodeName: w.conf.Node,
		},
	}

	// Loop while long polling.
	// Exponential backoff on registration errors.
	retryWait := w.conf.MinRetry
	for {
		println("registering...")
		resp, err := w.conn.Register(w.ctx, registerreq)
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
		}

		if resp.Directive == "reserved" {
			println("reserved...")
			// Note that exiting this loop will start
			// the zombie timer to os.Exit in case the reserving
			// root process does not start sending /worker/chucksearch.
			break
		}

		if resp.Directive != "reregister" {
			w.logger.Warn(
				"Unexpected registration response",
				zap.String("directive", resp.Directive),
				zap.String("selfaddr", w.selfaddr),
				zap.String("recruiteraddr", w.conf.Recruiter),
				zap.String("nodename", w.conf.Node),
			)
		}
		w.logger.Info("Reregister", zap.String("selfaddr", w.selfaddr))
	}
}

func (w *WorkerReg) StartZombieTimer() {
	// Note reset can fail if timer is not stopped,
	// and a stopped timer must be drained.
	// https://golang.org/pkg/time/#Timer.Reset
	if !w.zombieTimer.Stop() {
		<-w.zombieTimer.C
	}
	w.zombieTimer.Reset(w.zombieDuration)
}

func (w *WorkerReg) StopZombieTimer() {
	w.zombieTimer.Stop()
}

func (w *WorkerReg) zombieKiller() {
	<-w.zombieTimer.C
	w.logger.Info("Zombie Killer is terminating the process")
	os.Exit(0)
	// Note that a zombie process should exit, rather than try to recover,
	// because the zombie state is exceptional and should not happen unless
	// there was an unexpected failure. The failure is most likely in the
	// root zqd process, but this zombie *might* have been a contributing factor,
	// so it is safest to exit and restart.
}
