package storage

import (
	"context"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng/resolver"
)

type Summary struct {
	Kind        api.StorageKind
	Span        nano.Span
	DataBytes   int64
	RecordCount int64
}

type Storage interface {
	Kind() api.StorageKind
	NativeOrder() zbuf.Order
	Summary(ctx context.Context) (Summary, error)
	Write(ctx context.Context, zctx *resolver.Context, zr zbuf.Reader) error
}
