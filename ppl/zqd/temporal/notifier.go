package temporal

import (
	"context"
	"time"

	"github.com/brimsec/zq/api"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

type Notifier struct {
	client client.Client
	logger *zap.Logger
}

func NewNotifier(logger *zap.Logger, conf Config) (*Notifier, error) {
	client, err := NewClient(logger, conf)
	return &Notifier{client, logger}, err
}

func (n *Notifier) Shutdown() {
	n.client.Close()
}

func (n *Notifier) SpaceDeleted(ctx context.Context, id api.SpaceID) {
	if err := n.client.CancelWorkflow(ctx, id.String(), ""); err != nil {
		n.logger.Warn("Canceling workflow failed", zap.Error(err), zap.Stringer("space_id", id))
	}
}

func (n *Notifier) SpaceWritten(ctx context.Context, id api.SpaceID) {
	opts := client.StartWorkflowOptions{
		ID:        id.String(),
		TaskQueue: TaskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
		},
	}
	_, err := n.client.SignalWithStartWorkflow(ctx, id.String(), writeSignal, "", opts, spaceWorkflow)
	if err != nil {
		n.logger.Warn("Signaling workflow failed", zap.Error(err), zap.Stringer("space_id", id))
	}
}
