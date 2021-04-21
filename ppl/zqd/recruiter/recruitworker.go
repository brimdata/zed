// This is the client-side layer that is used by the zqd root process during execution of a query.
// It is called within the driver package.
package recruiter

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/ppl/zqd/worker"
	"github.com/brimdata/zed/proc"
	"go.uber.org/zap"
)

// RecruitWorkers is used by the zqd root process to recruit workers for a distributed query.
func RecruitWorkers(ctx *proc.Context, workerCount int, conf worker.WorkerConfig, logger *zap.Logger) ([]string, error) {
	logger.Info("Recruit workers", zap.Int("count", workerCount))
	if conf.BoundWorkers != "" {
		// BoundWorkers is a fixed list of workers bound to a root process.
		// It is used for ZTests and simple clusters without a recruiter.
		workers := strings.Split(conf.BoundWorkers, ",")
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
	recreq := api.RecruitRequest{NumberRequested: workerCount, Label: conf.Host}
	resp, err := conn.Recruit(ctx, recreq)
	if err != nil {
		return nil, fmt.Errorf("error on recruit for recruiter at %s : %w", conf.Recruiter, err)
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
