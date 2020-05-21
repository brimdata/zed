package storage

import (
	"context"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

type Summary struct {
	Span      nano.Span
	DataBytes int64
}

type Storage interface {
	Open(ctx context.Context, span nano.Span) (zbuf.ReadCloser, error)
	Summary(ctx context.Context) (Summary, error)
	NativeDirection() zbuf.Direction
}
