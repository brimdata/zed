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
}

func NewWorkerReg(ctx context.Context, srvAddr string, recruiteraddr string, specPodIP string, specNodeName string) (*WorkerReg, error) {
	host, port, _ := net.SplitHostPort(srvAddr)
	if specPodIP != "" {
		host = specPodIP
	}
	return &WorkerReg{
		conn:          client.NewConnectionTo("http://" + recruiteraddr),
		nodename:      specNodeName,
		recruiteraddr: recruiteraddr,
		selfaddr:      net.JoinHostPort(host, port),
	}, nil
}

func (w *WorkerReg) RegisterWithRecruiter(ctx context.Context, logger *zap.Logger) error {
	unreservereq := api.UnreserveRequest{
		Addrs: []string{w.selfaddr},
	}
	resp1, err := w.conn.Unreserve(ctx, unreservereq)
	if err != nil {
		return fmt.Errorf("error on unreserve with recruiter at %s : %v", w.recruiteraddr, err)
	}
	if resp1.Reserved != false {
		return fmt.Errorf("recruiter did not acknowlege unreserve")
	}

	registerreq := api.RegisterRequest{
		Worker: api.Worker{
			Addr:     w.selfaddr,
			NodeName: w.nodename,
		},
	}
	resp2, err := w.conn.Register(ctx, registerreq)
	if err != nil {
		return fmt.Errorf("error on register with recruiter at %s : %v", w.recruiteraddr, err)
	}
	if resp2.Registered != true {
		return fmt.Errorf("recruiter did not acknowlege register")
	}
	logger.Info(
		"Registered",
		zap.String("selfaddr", w.selfaddr),
		zap.String("recruiteraddr", w.recruiteraddr),
		zap.String("nodename", w.nodename),
	)
	return nil
}
