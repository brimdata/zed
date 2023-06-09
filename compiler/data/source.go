package data

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/segmentio/ksuid"
)

type Source struct {
	engine storage.Engine
	lake   *lake.Root
}

func NewSource(engine storage.Engine, lake *lake.Root) *Source {
	return &Source{
		engine: engine,
		lake:   lake,
	}
}

func (s *Source) IsLake() bool {
	return s.lake != nil
}

func (s *Source) Lake() *lake.Root {
	return s.lake
}

func (s *Source) PoolID(ctx context.Context, id string) (ksuid.KSUID, error) {
	if s.lake != nil {
		return s.lake.PoolID(ctx, id)
	}
	return ksuid.Nil, nil
}

func (s *Source) CommitObject(ctx context.Context, id ksuid.KSUID, name string) (ksuid.KSUID, error) {
	if s.lake != nil {
		return s.lake.CommitObject(ctx, id, name)
	}
	return ksuid.Nil, nil
}

func (s *Source) SortKey(ctx context.Context, src dag.Op) order.SortKey {
	if s.lake != nil {
		return s.lake.SortKey(ctx, src)
	}
	return order.Nil
}

func (s *Source) Open(ctx context.Context, zctx *zed.Context, path, format string, pushdown zbuf.Filter) (zbuf.Puller, error) {
	if path == "-" {
		path = "stdio:stdin"
	}
	file, err := anyio.Open(ctx, zctx, s.engine, path, anyio.ReaderOpts{Format: format})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	scanner, err := zbuf.NewScanner(ctx, file, pushdown)
	if err != nil {
		file.Close()
		return nil, err
	}
	sn := zbuf.NamedScanner(scanner, path)
	return &closePuller{sn, file}, nil
}

func (s *Source) OpenHTTP(ctx context.Context, zctx *zed.Context, url, format, method string, headers http.Header, body io.Reader) (zbuf.Puller, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = headers
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	file, err := anyio.NewFile(zctx, resp.Body, url, anyio.ReaderOpts{Format: format})
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	scanner, err := zbuf.NewScanner(ctx, file, nil)
	if err != nil {
		file.Close()
		return nil, err
	}
	return &closePuller{scanner, file}, nil
}

type closePuller struct {
	p zbuf.Puller
	c io.Closer
}

func (c *closePuller) Pull(done bool) (zbuf.Batch, error) {
	batch, err := c.p.Pull(done)
	if batch == nil {
		c.c.Close()
	}
	return batch, err
}
