package recruiter

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/proc"
)

func RecruitWorkers(pctx *proc.Context, workerCount int) ([]string, error) {
	// ZQD_TEST_WORKERS is used for Ztests
	if workerstr := os.Getenv("ZQD_TEST_WORKERS"); workerstr != "" {
		workers := strings.Split(workerstr, ",")
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

	var raddr string
	if raddr = os.Getenv("ZQD_RECRUIT"); raddr == "" {
		return nil, fmt.Errorf("distributed exec failure: ZQD_RECRUIT not present")
	}
	if _, _, err := net.SplitHostPort(raddr); err != nil {
		return nil, fmt.Errorf("distributed exec failure: ZQD_RECRUIT for root process does not have host:port")
	}
	conn := client.NewConnectionTo("http://" + raddr)
	recreq := api.RecruitRequest{NumberRequested: workerCount}
	resp, err := conn.Recruit(pctx, recreq)
	if err != nil {
		return nil, fmt.Errorf("distributed exec failure: error on recruit for recruiter at %s : %v", raddr, err)
	}
	if workerCount > len(resp.Workers) {
		// TODO: we should fail back to running the query with fewer workers if possible.
		// Determining when that is possible is non-trivial.
		// Alternative is to wait and try to recruit more workers,
		// which would reserve the idle zqd root process while waiting. -MTW
		return nil, fmt.Errorf("distributed exec failure: requested workers %d greater than available workers %d",
			workerCount, len(resp.Workers))
	}

	var workers []string
	for _, w := range resp.Workers {
		workers = append(workers, w.Addr)
	}
	return workers, nil
}

func ReleaseWorker(ctx context.Context, conn *client.Connection) error {
	return conn.WorkerRelease(ctx)
}
