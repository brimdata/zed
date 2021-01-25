// This is the client-side layer that is used by the zqd root process during execution of a query.
// It is called within the driver package.
package recruiter

import (
	"fmt"
	"net"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/ppl/zqd/worker"
	"github.com/brimsec/zq/proc"
	"go.uber.org/zap"
)

func GetWorkerConnection(pctx *proc.Context, conf worker.WorkerConfig, logger *zap.Logger) (*client.Connection, error) {
	workers, err := recruitWorkers(pctx, 1, conf, logger)
	if err != nil {
		return nil, err
	}
	if len(workers) == 0 {
		return nil, fmt.Errorf("no worker is available")
	}
	return client.NewConnectionTo("http://" + workers[0]), nil
}

func recruitWorkers(ctx *proc.Context, workerCount int, conf worker.WorkerConfig, logger *zap.Logger) ([]string, error) {
	logger.Info("Recruit workers", zap.Int("count", workerCount))
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
		return nil, fmt.Errorf("error on recruit for recruiter at %s : %v", conf.Recruiter, err)
	}
	var workers []string
	for _, w := range resp.Workers {
		workers = append(workers, w.Addr)
	}
	return workers, nil
}
