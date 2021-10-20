package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/segmentio/ksuid"
)

type FileAdaptor struct {
	engine storage.Engine
}

var _ proc.DataAdaptor = (*FileAdaptor)(nil)

func NewFileAdaptor(engine storage.Engine) *FileAdaptor {
	return &FileAdaptor{
		engine: engine,
	}
}

func (*FileAdaptor) PoolID(context.Context, string) (ksuid.KSUID, error) {
	return ksuid.Nil, nil
}

func (*FileAdaptor) CommitObject(context.Context, ksuid.KSUID, string) (ksuid.KSUID, error) {
	return ksuid.Nil, nil
}

func (*FileAdaptor) Layout(context.Context, dag.Source) order.Layout {
	return order.Nil
}

func (*FileAdaptor) NewScheduler(context.Context, *zed.Context, dag.Source, extent.Span, zbuf.Filter, []dag.IndexPredicate) (proc.Scheduler, error) {
	return nil, errors.New("pool scan not available when running on local file system")
}

func (f *FileAdaptor) Open(ctx context.Context, zctx *zed.Context, path string, pushdown zbuf.Filter) (zbuf.PullerCloser, error) {
	if path == "-" {
		path = "stdio:stdin"
	}
	file, err := anyio.Open(ctx, zctx, f.engine, path, anyio.ReaderOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	scanner, err := zbuf.NewScanner(ctx, file, pushdown)
	if err != nil {
		file.Close()
		return nil, err
	}
	sn := zbuf.NamedScanner(scanner, path)
	return &struct {
		zbuf.Scanner
		io.Closer
	}{sn, file}, nil
}
