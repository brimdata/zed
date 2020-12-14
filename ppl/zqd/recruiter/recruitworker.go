package recruiter

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/proc"
	"go.uber.org/zap"
)

func RecruitWorkers(ctx *proc.Context, workerCount int, conf WorkerConfig, logger *zap.Logger) ([]string, error) {
	logger.Info("RecruitWorkers", zap.Int("workerCount", workerCount))
	if conf.BoundWorkers != "" {
		// Special case: workerstr is used for ZTests
		workers := strings.Split(conf.BoundWorkers, ",")
		if workerCount > len(workers) {
			return nil, fmt.Errorf("requested parallelism %d is greater than the number of workers %d",
				workerCount, len(workers))
		}
		for _, w := range workers {
			if _, _, err := net.SplitHostPort(w); err != nil {
				return nil, err
			}
		}
		return workers, nil
	}

	if conf.Recruiter == "" {
		return nil, fmt.Errorf("flag -worker.recruiter is not present")
	}
	if _, _, err := net.SplitHostPort(conf.Recruiter); err != nil {
		return nil, fmt.Errorf("flag -worker.recruiter does not have host:port")
	}
	conn := client.NewConnectionTo("http://" + conf.Recruiter)
	recreq := api.RecruitRequest{NumberRequested: workerCount}
	resp, err := conn.Recruit(ctx, recreq)
	if err != nil {
		return nil, fmt.Errorf("error on recruit for recruiter at %s : %v", conf.Recruiter, err)
	}
	if workerCount > len(resp.Workers) {
		err := fmt.Errorf("requested workers %d greater than available workers %d",
			workerCount, len(resp.Workers))
		if !conf.Fallback {
			return nil, err
		}
		logger.Warn("Worker fallback", zap.Error(err))
	}

	var workers []string
	for _, w := range resp.Workers {
		workers = append(workers, w.Addr)
	}
	return workers, nil
}

func ReleaseWorker(ctx context.Context, conn *client.Connection, logger *zap.Logger) error {
	logger.Info("ReleaseWorker", zap.String("addr", conn.ClientHostURL()))
	return conn.WorkerRelease(ctx)
}
