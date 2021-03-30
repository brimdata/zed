package storage

import (
	"context"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng/resolver"
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
