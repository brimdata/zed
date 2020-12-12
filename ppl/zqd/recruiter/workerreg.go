package recruiter

import (
	"context"
	"fmt"
	"net"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"go.uber.org/zap"
)

type WorkerReg struct {
	conn          *client.Connection
	recruiteraddr string
	selfaddr      string
	nodename      string
	timeout       int
}

func NewWorkerReg(ctx context.Context, srvAddr string, recruiteraddr string, workerHost string, workerNode string, workerTimeout int) (*WorkerReg, error) {
	host, port, _ := net.SplitHostPort(srvAddr)
	if workerHost != "" {
		host = workerHost
	}
	return &WorkerReg{
		conn:          client.NewConnectionTo("http://" + recruiteraddr),
		nodename:      workerNode,
		recruiteraddr: recruiteraddr,
		selfaddr:      net.JoinHostPort(host, port),
		timeout:       workerTimeout,
	}, nil
}

func (w *WorkerReg) RegisterWithRecruiter(ctx context.Context, logger *zap.Logger) error {
	// This should be a loop that tries to reregister, called as a goroutine.
	// Loop should be suspended when a /worker/search is in progress, and
	// resume afterwards.
	// So, break out of loop when reserved, then register is called again on /worker/release
	// Failure case is when /worker/release is not called. Maybe we need some locks and timers
	// to take care of that.
	registerreq := api.RegisterRequest{
		Timeout: w.timeout,
		Worker: api.Worker{
			Addr:     w.selfaddr,
			NodeName: w.nodename,
		},
	}
	// this will be a long poll:
	resp, err := w.conn.Register(ctx, registerreq)
	if err != nil {
		return fmt.Errorf("error on register with recruiter at %s : %v", w.recruiteraddr, err)
	}

	// various logic based on directive here

	logger.Info(
		"Registered response",
		zap.String("directive", resp.Directive),
		zap.String("selfaddr", w.selfaddr),
		zap.String("recruiteraddr", w.recruiteraddr),
		zap.String("nodename", w.nodename),
	)
	return nil
}
