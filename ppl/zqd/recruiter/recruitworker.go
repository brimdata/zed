// This is the client-side layer that is used by the zqd root process during execution of a query.
// It is called within the driver package.
package recruiter

import (
	"fmt"
	"net"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/ppl/zqd/worker"
	"github.com/brimsec/zq/proc"
	"go.uber.org/zap"
)

// GetWorkerConnection should only wait in unusual circumstances.
// Here is the reasoning:
// (1) In normal operation, we would expect it to return immediately,
// and not retry, since there should be a queue of workers in a
// "registered" state (long-polling).
// (2) We should tune our EKS cluster so its limiting factor is
// bandwidth to S3, not memory or CPU. So for a properly tuned cluster,
// there should be workers always available, and they should be
// requested again as they complete their S3 reads.
// (3) The retry loop here is so we will fail gracefully when the EKS
// cluster is poorly tuned, and in order to support tests where
// not enough workers are available.
// (4) We use a retry with exponential backoff because the Recruiter
// process is already maintaining long-lived connections with
// available workers, so it does not make sense to also have the
// Recruiter simultaneously maintain long connections with
// root processes that are requesting workers.
func GetWorkerConnection(pctx *proc.Context, conf worker.WorkerConfig, logger *zap.Logger) (*client.Connection, error) {
	var workers []string
	retryWait := conf.MinRetry
	for len(workers) == 0 {
		var err error
		workers, err = recruitWorkers(pctx, 1, conf, logger)
		if err != nil {
			return nil, err
		}
		logger.Info("Retrying recruit worker")
		time.Sleep(retryWait)
		if retryWait < conf.MaxRetry {
			retryWait = (retryWait * 3) / 2
		} else {
			break
		}
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
