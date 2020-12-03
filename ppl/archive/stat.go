package archive

import (
	"context"
	"errors"

	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/chunk"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

// statReadCloser implements zbuf.ReadCloser.
type statReadCloser struct {
	ctx    context.Context
	cancel context.CancelFunc
	ark    *Archive
	zctx   *resolver.Context
	recs   chan *zng.Record
	err    error

	chunkBuilder  *zng.Builder
	indexBuilders map[string]*zng.Builder
}

func (s *statReadCloser) Read() (*zng.Record, error) {
	select {
	case r, ok := <-s.recs:
		if !ok {
			return nil, s.err
		}
		return r, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *statReadCloser) Close() error {
	s.cancel()
	return nil
}

func (s *statReadCloser) chunkRecord(chunk chunk.Chunk) error {
	fi, err := iosrc.Stat(s.ctx, chunk.Path())
	if err != nil {
		return err
	}

	if s.chunkBuilder == nil {
		s.chunkBuilder = zng.NewBuilder(s.zctx.MustLookupTypeRecord([]zng.Column{
			zng.NewColumn("type", zng.TypeString),
			zng.NewColumn("log_id", zng.TypeString),
			zng.NewColumn("first", zng.TypeTime),
			zng.NewColumn("last", zng.TypeTime),
			zng.NewColumn("size", zng.TypeUint64),
			zng.NewColumn("record_count", zng.TypeUint64),
		}))
	}

	rec := s.chunkBuilder.Build(
		zng.EncodeString("chunk"),
		zng.EncodeString(s.ark.Root.RelPath(chunk.Path())),
		zng.EncodeTime(chunk.First),
		zng.EncodeTime(chunk.Last),
		zng.EncodeUint(uint64(fi.Size())),
		zng.EncodeUint(uint64(chunk.RecordCount)),
	).Keep()
	select {
	case s.recs <- rec:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *statReadCloser) indexRecord(chunk chunk.Chunk, indexPath string) error {
	zardir := chunk.ZarDir()
	info, err := microindex.Stat(s.ctx, zardir.AppendPath(indexPath))
	if err != nil {
		if errors.Is(err, zqe.E(zqe.NotFound)) {
			return nil
		}
		return err
	}

	if s.indexBuilders == nil {
		s.indexBuilders = make(map[string]*zng.Builder)
	}
	if s.indexBuilders[indexPath] == nil {
		keycols := make([]zng.Column, len(info.Keys))
		for i, k := range info.Keys {
			keycols[i] = zng.Column{
				Name: k.Name,
				Type: zng.TypeString,
			}
		}
		keyrec := s.zctx.MustLookupTypeRecord(keycols)

		s.indexBuilders[indexPath] = zng.NewBuilder(s.zctx.MustLookupTypeRecord([]zng.Column{
			zng.NewColumn("type", zng.TypeString),
			zng.NewColumn("log_id", zng.TypeString),
			zng.NewColumn("first", zng.TypeTime),
			zng.NewColumn("last", zng.TypeTime),
			zng.NewColumn("index_id", zng.TypeString),
			zng.NewColumn("size", zng.TypeUint64),
			zng.NewColumn("record_count", zng.TypeUint64),
			zng.NewColumn("keys", keyrec),
		}))
	}

	if len(s.indexBuilders[indexPath].Type.Columns) != 7+len(info.Keys) {
		return zqe.E("key record differs in index files %s %s", indexPath, chunk)
	}
	var keybytes zcode.Bytes
	for _, k := range info.Keys {
		keybytes = zcode.AppendPrimitive(keybytes, zng.EncodeString(k.TypeName))
	}

	rec := s.indexBuilders[indexPath].Build(
		zng.EncodeString("index"),
		zng.EncodeString(s.ark.Root.RelPath(chunk.Path())),
		zng.EncodeTime(chunk.First),
		zng.EncodeTime(chunk.Last),
		zng.EncodeString(indexPath),
		zng.EncodeUint(uint64(info.Size)),
		zng.EncodeUint(uint64(chunk.RecordCount)),
		keybytes,
	).Keep()
	select {
	case s.recs <- rec:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *statReadCloser) run() {
	defer close(s.recs)

	s.err = Walk(s.ctx, s.ark, func(chunk chunk.Chunk) error {
		if err := s.chunkRecord(chunk); err != nil {
			return err
		}
		if dirents, err := s.ark.dataSrc.ReadDir(s.ctx, chunk.ZarDir()); err == nil {
			for _, e := range dirents {
				if e.IsDir() {
					continue
				}
				if err := s.indexRecord(chunk, e.Name()); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func RecordCount(ctx context.Context, ark *Archive) (uint64, error) {
	var count uint64
	err := Walk(ctx, ark, func(chunk chunk.Chunk) error {
		count += chunk.RecordCount
		return nil
	})
	return count, err
}

func Stat(ctx context.Context, zctx *resolver.Context, ark *Archive) (zbuf.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &statReadCloser{
		ctx:    ctx,
		cancel: cancel,
		ark:    ark,
		zctx:   zctx,
		recs:   make(chan *zng.Record),
	}
	go s.run()
	return s, nil
}
