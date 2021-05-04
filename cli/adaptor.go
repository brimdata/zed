package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type FileAdaptor struct {
	ctx  context.Context
	zctx *zson.Context
}

var _ proc.DataAdaptor = (*FileAdaptor)(nil)

func NewFileAdaptor(ctx context.Context, zctx *zson.Context) *FileAdaptor {
	return &FileAdaptor{
		ctx:  ctx,
		zctx: zctx,
	}
}

func (f *FileAdaptor) Lookup(_ context.Context, _ string) (ksuid.KSUID, error) {
	return ksuid.Nil, nil
}

func (f *FileAdaptor) LayoutOf(_ context.Context, _ ksuid.KSUID) (order.Layout, error) {
	return order.Nil, errors.New("pool scan not available when running on local file system")
}

func (f *FileAdaptor) NewScheduler(_ context.Context, _ *zson.Context, _ *dag.Pool, pushdown zbuf.Filter) (proc.Scheduler, error) {
	return nil, errors.New("pool scan not available when running on local file system")
}

func (f *FileAdaptor) Open(_ context.Context, _ *zson.Context, path string, pushdown zbuf.Filter) (zbuf.PullerCloser, error) {
	if path == "-" {
		path = iosrc.Stdin
	}
	file, err := anyio.OpenFile(f.zctx, path, anyio.ReaderOpts{})
	if err != nil {
		err = fmt.Errorf("%s: %w", path, err)
		fmt.Fprintln(os.Stderr, err)
	}
	scanner, err := zbuf.NewScanner(f.ctx, file, pushdown, nano.MaxSpan)
	if err != nil {
		return nil, err
	}
	sn := zbuf.NamedScanner(scanner, path)
	return zbuf.ScannerNopCloser(sn), nil
}

func (*FileAdaptor) Get(_ context.Context, _ *zson.Context, url string, pushdown zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, errors.New("http source not yet implemented")
}
