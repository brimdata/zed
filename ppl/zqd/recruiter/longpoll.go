package recruiter

import (
	"context"
	"time"

	"github.com/brimdata/zed/api"
	"go.uber.org/zap"
)

func RecruitWithEndWait(wp *WorkerPool, numberRequested int, loggingLabel string, logger *zap.Logger) ([]api.Worker, error) {
	ws, err := wp.Recruit(numberRequested)
	if err != nil {
		return nil, err
	}
	var workers []api.Worker
	for _, w := range ws {
		if w.Callback(RecruitmentDetail{LoggingLabel: loggingLabel, NumberRequested: numberRequested}) {
			workers = append(workers, api.Worker{Addr: w.Addr, NodeName: w.NodeName})
		}
	}
	logger.Info("Recruit request",
		zap.String("label", loggingLabel),
		zap.Int("requested", numberRequested),
		zap.Int("recruited", len(workers)))
	return workers, nil
}

func WaitForRecruitment(ctx context.Context, wp *WorkerPool, addr string, nodename string, timeout int, logger *zap.Logger) (string, bool, error) {
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()
	recruited := make(chan RecruitmentDetail)
	if err := wp.Register(addr, nodename, func(rd RecruitmentDetail) bool {
		select {
		case recruited <- rd:
		default:
			// If the receiver is not ready it means the worker has already
			// Deregistered and is unavailable.
			logger.Warn("Receiver not ready for recruited", zap.String("label", rd.LoggingLabel))
			return false
		}
		return true
	}); err != nil {
		return "", false, err
	}
	var directive string
	select {
	case rd := <-recruited:
		logger.Info("Worker recruited",
			zap.String("addr", addr),
			zap.String("label", rd.LoggingLabel),
			zap.Int("count", rd.NumberRequested))
		directive = "reserved"
	case <-timer.C:
		logger.Debug("Worker should reregister", zap.String("addr", addr))
		wp.Deregister(addr)
		directive = "reregister"
	case <-ctx.Done():
		logger.Info("HandleRegister context cancel")
		wp.Deregister(addr)
		return "", true, nil
	}
	return directive, false, nil
}
