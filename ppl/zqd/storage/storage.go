package storage

import (
	"context"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
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
