package archivestore

import (
	"context"
	"sync"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqe"
)

func Load(ctx context.Context, path iosrc.URI, cfg *storage.ArchiveConfig) (*Storage, error) {
	co := &archive.CreateOptions{}
	if cfg != nil && cfg.CreateOptions != nil {
		co.LogSizeThreshold = cfg.CreateOptions.LogSizeThreshold
	}
	oo := &archive.OpenOptions{}
	if cfg != nil && cfg.OpenOptions != nil {
		oo.LogFilter = cfg.OpenOptions.LogFilter
	}
	ark, err := archive.CreateOrOpenArchiveWithContext(ctx, path.String(), co, oo)
	if err != nil {
		return nil, err
	}
	return &Storage{ark: ark}, nil
}

type summaryCache struct {
	mu          sync.Mutex
	lastUpdate  int
	span        nano.Span
	dataBytes   int64
	recordCount int64
}

type Storage struct {
	ark *archive.Archive
}

func (s *Storage) NativeDirection() zbuf.Direction {
	return s.ark.DataSortDirection
}

func (s *Storage) MultiSource() driver.MultiSource {
	return archive.NewMultiSource(s.ark, nil)
}

func (s *Storage) Summary(ctx context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = storage.ArchiveStore
	err := archive.SpanWalk(ctx, s.ark, func(si archive.SpanInfo, zardir iosrc.URI) error {
		zngpath := archive.ZarDirToLog(zardir)
		info, err := iosrc.Stat(ctx, zngpath)
		if err != nil {
			return err
		}
		sum.DataBytes += info.Size()
		sum.RecordCount += int64(si.RecordCount)
		if sum.Span.Dur == 0 {
			sum.Span = si.Span()
		} else {
			sum.Span = sum.Span.Union(si.Span())
		}
		return nil
	})
	return sum, err
}

func (s *Storage) Write(ctx context.Context, zctx *resolver.Context, zr zbuf.Reader) error {
	return archive.Import(ctx, s.ark, zctx, zr)
}

func (s *Storage) IndexCreate(ctx context.Context, req api.IndexPostRequest) error {
	var rules []archive.Rule
	if req.AST != nil {
		proc, err := ast.UnpackJSON(nil, req.AST)
		if err != nil {
			return zqe.E(zqe.Invalid, err)
		}
		rule, err := archive.NewRuleAST("zql", proc, req.OutputFile, req.Keys, 0)
		if err != nil {
			return zqe.E(zqe.Invalid, err)
		}
		rules = append(rules, *rule)
	}
	for _, pattern := range req.Patterns {
		rule, err := archive.NewRule(pattern)
		if err != nil {
			return zqe.E(zqe.Invalid, err)
		}
		rules = append(rules, *rule)
	}
	inputFile := req.InputFile
	if inputFile == "" {
		inputFile = "_"
	}
	// XXX Eventually this method should provide progress updates.
	return archive.IndexDirTree(ctx, s.ark, rules, inputFile, nil)
}

func (s *Storage) IndexSearch(ctx context.Context, zctx *resolver.Context, query archive.IndexQuery) (zbuf.ReadCloser, error) {
	return archive.FindReadCloser(ctx, zctx, s.ark, query, archive.AddPath(archive.DefaultAddPathField, false))
}

func (s *Storage) ArchiveStat(ctx context.Context, zctx *resolver.Context) (zbuf.ReadCloser, error) {
	return archive.Stat(ctx, zctx, s.ark)
}
