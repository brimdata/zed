package archivestore

import (
	"context"
	"sync"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/storage"
)

func Load(path iosrc.URI, cfg *storage.ArchiveConfig) (*Storage, error) {
	co := &archive.CreateOptions{}
	if cfg != nil && cfg.CreateOptions != nil {
		co.LogSizeThreshold = cfg.CreateOptions.LogSizeThreshold
	}
	oo := &archive.OpenOptions{}
	if cfg != nil && cfg.OpenOptions != nil {
		oo.LogFilter = cfg.OpenOptions.LogFilter
	}
	ark, err := archive.CreateOrOpenArchive(path.String(), co, oo)
	if err != nil {
		return nil, err
	}
	return &Storage{ark: ark}, nil
}

type summaryCache struct {
	mu         sync.Mutex
	lastUpdate int
	span       nano.Span
	dataBytes  int64
}

type Storage struct {
	ark      *archive.Archive
	sumCache summaryCache
}

func (s *Storage) NativeDirection() zbuf.Direction {
	return s.ark.DataSortDirection
}

func (s *Storage) Open(ctx context.Context, zctx *resolver.Context, span nano.Span) (zbuf.ReadCloser, error) {
	var err error
	var paths []string
	err = archive.SpanWalk(s.ark, func(si archive.SpanInfo, zardir iosrc.URI) error {
		if span.Overlaps(si.Span) {
			p := archive.ZarDirToLog(zardir)
			// XXX Doing this because detector doesn't support file uri's. At
			// some point it should.
			if p.Scheme == "file" {
				paths = append(paths, p.Filepath())
			} else {
				paths = append(paths, p.String())
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	cfg := detector.OpenConfig{Format: "zng"}
	return detector.MultiFileReader(zctx, paths, cfg), nil
}

func (s *Storage) Summary(_ context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = storage.ArchiveStore

	update, err := s.ark.UpdateCheck()
	if err != nil {
		return sum, err
	}

	s.sumCache.mu.Lock()
	if update == s.sumCache.lastUpdate {
		sum.Span = s.sumCache.span
		sum.DataBytes = s.sumCache.dataBytes
		s.sumCache.mu.Unlock()
		return sum, nil
	}
	s.sumCache.mu.Unlock()

	err = archive.SpanWalk(s.ark, func(si archive.SpanInfo, zardir iosrc.URI) error {
		zngpath := archive.ZarDirToLog(zardir)
		info, err := iosrc.Stat(zngpath)
		if err != nil {
			return err
		}
		sum.DataBytes += info.Size()
		if sum.Span.Dur == 0 {
			sum.Span = si.Span
		} else {
			sum.Span = sum.Span.Union(si.Span)
		}
		return nil
	})
	if err != nil {
		return sum, err
	}

	s.sumCache.mu.Lock()
	s.sumCache.lastUpdate = update
	s.sumCache.span = sum.Span
	s.sumCache.dataBytes = sum.DataBytes
	s.sumCache.mu.Unlock()

	return sum, nil
}

func (s *Storage) Write(ctx context.Context, zctx *resolver.Context, zr zbuf.Reader) error {
	return archive.Import(ctx, s.ark, zctx, zr)
}

func (s *Storage) IndexSearch(ctx context.Context, zctx *resolver.Context, query archive.IndexQuery) (zbuf.ReadCloser, error) {
	return archive.FindReadCloser(ctx, zctx, s.ark, query, archive.AddPath(archive.DefaultAddPathField, false))
}

func (s *Storage) ArchiveStat(ctx context.Context, zctx *resolver.Context) (zbuf.ReadCloser, error) {
	return archive.Stat(ctx, zctx, s.ark)
}
