package archivestore

import (
	"context"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/storage"
)

func Load(path string) (*Storage, error) {
	ark, err := archive.OpenArchive(path)
	if err != nil {
		return nil, err
	}
	return &Storage{ark: ark}, nil
}

type Storage struct {
	ark *archive.Archive
}

func (s *Storage) NativeDirection() zbuf.Direction {
	return s.ark.Meta.DataSortDirection
}

func (s *Storage) Open(ctx context.Context, span nano.Span) (zbuf.ReadCloser, error) {
	var err error
	var paths []string
	err = archive.SpanWalk(s.ark, func(si archive.SpanInfo, zardir string) error {
		if span.Overlaps(si.Span) {
			paths = append(paths, archive.ZarDirToLog(zardir))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	zctx := resolver.NewContext()
	cfg := detector.OpenConfig{Format: "zng"}
	return detector.MultiFileReader(zctx, paths, cfg), nil
}

func (s *Storage) Summary(_ context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = storage.ArchiveStore
	return sum, archive.SpanWalk(s.ark, func(si archive.SpanInfo, zardir string) error {
		zngpath := archive.ZarDirToLog(zardir)
		sinfo, err := os.Stat(zngpath)
		if err != nil {
			return err
		}
		sum.DataBytes += sinfo.Size()
		if sum.Span.Dur == 0 {
			sum.Span = si.Span
		} else {
			sum.Span = sum.Span.Union(si.Span)
		}
		return nil
	})
}

type indexSearch struct {
	ctx    context.Context
	cancel context.CancelFunc
	hits   chan *zng.Record
	err    error
}

func (is *indexSearch) Read() (*zng.Record, error) {
	select {
	case r, ok := <-is.hits:
		if !ok {
			return nil, is.err
		}
		return r, nil
	case <-is.ctx.Done():
		return nil, is.ctx.Err()
	}
}

func (is *indexSearch) Close() error {
	is.cancel()
	return nil
}

func (s *Storage) IndexSearch(ctx context.Context, query archive.IndexQuery) (zbuf.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	is := &indexSearch{
		ctx:    ctx,
		cancel: cancel,
		hits:   make(chan *zng.Record),
	}
	go func() {
		is.err = archive.Find(ctx, s.ark, query, is.hits, archive.AddPath(archive.DefaultAddPathField, false))
		close(is.hits)
	}()
	return is, nil
}
