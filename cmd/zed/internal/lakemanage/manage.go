package lakemanage

import (
	"context"
	"errors"
	"syscall"
	"time"

	"github.com/brimdata/zed/api/client"
	lakeapi "github.com/brimdata/zed/lake/api"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Update(ctx context.Context, lk lakeapi.Interface, conf Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	branches, err := getBranches(ctx, conf, lk, logger)
	if err != nil {
		return err
	}
	group, ctx := errgroup.WithContext(ctx)
	for _, branch := range branches {
		branch.logger.Info("updating pool")
		if err := branch.run(ctx); err != nil {
			branch.logger.Error("update error", zap.Error(err))
		}
	}
	return group.Wait()
}

func Monitor(ctx context.Context, conn *client.Connection, conf Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}
	logger.Info("monitoring")
	lk := lakeapi.NewRemoteLake(conn)
	for {
		err := monitor(ctx, lk, conf, logger)
		if errors.Is(err, syscall.ECONNREFUSED) {
			logger.Info("cannot connect to lake, retrying in 5 seconds")
		} else if err != nil {
			return err
		}
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func monitor(ctx context.Context, lk lakeapi.Interface, conf Config, logger *zap.Logger) error {
	for {
		select {
		case <-time.After(conf.interval()):
		case <-ctx.Done():
			return ctx.Err()
		}
		err := Update(ctx, lk, conf, logger)
		if err != nil {
			return err
		}
	}
}

func getBranches(ctx context.Context, conf Config, lk lakeapi.Interface, logger *zap.Logger) ([]*branch, error) {
	pools, err := lakeapi.GetPools(ctx, lk)
	if err != nil {
		return nil, err
	}
	var branches []*branch
	for _, pool := range pools {
		if b := newBranch(conf, pool, lk, logger); b != nil {
			branches = append(branches, b)
		}
	}
	return branches, nil
}
