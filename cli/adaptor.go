package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zson"
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

func (f *FileAdaptor) Lookup(_ context.Context, _ string) (ksuid.KSUID, error) {
	return ksuid.Nil, nil
}

func (f *FileAdaptor) Layout(_ context.Context, _ ksuid.KSUID) (order.Layout, error) {
	return order.Nil, errors.New("pool scan not available when running on local file system")
}

func (f *FileAdaptor) NewScheduler(_ context.Context, _ *zson.Context, _ *dag.Pool, pushdown zbuf.Filter) (proc.Scheduler, error) {
	return nil, errors.New("pool scan not available when running on local file system")
}

func (f *FileAdaptor) Open(ctx context.Context, zctx *zson.Context, path string, pushdown zbuf.Filter) (zbuf.PullerCloser, error) {
	if path == "-" {
		path = "stdio:stdin"
	}
	file, err := anyio.OpenFile(zctx, f.engine, path, anyio.ReaderOpts{})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	scanner, err := zbuf.NewScanner(ctx, file, pushdown, nano.MaxSpan)
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

func (f *FileAdaptor) Get(ctx context.Context, zctx *zson.Context, url string, pushdown zbuf.Filter) (zbuf.PullerCloser, error) {
	return f.Open(ctx, zctx, url, pushdown)
}
