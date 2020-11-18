package archivestore

import (
	"context"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/archive"
	"github.com/brimsec/zq/ppl/zqd/storage"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

func Load(ctx context.Context, path iosrc.URI, notifier WriteNotifier, cfg *api.ArchiveConfig) (*Storage, error) {
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
	return &Storage{
		ark:      ark,
		notifier: notifier,
	}, nil
}

type WriteNotifier interface {
	WriterClosed()
}

type summaryCache struct {
	mu          sync.Mutex
	lastUpdate  int
	span        nano.Span
	dataBytes   int64
	recordCount int64
}

type Storage struct {
	ark      *archive.Archive
	notifier WriteNotifier
}

func NewStorage(ark *archive.Archive) *Storage {
	return &Storage{ark: ark}
}

func (s *Storage) NativeOrder() zbuf.Order {
	return s.ark.DataOrder
}

func (s *Storage) MultiSource() driver.MultiSource {
	return archive.NewMultiSource(s.ark, nil)
}

func (s *Storage) StaticSource(src driver.Source) driver.MultiSource {
	return archive.NewStaticSource(s.ark, src)
}

func (s *Storage) Summary(ctx context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = api.ArchiveStore
	err := archive.Walk(ctx, s.ark, func(chunk archive.Chunk) error {
		zngpath := chunk.Path(s.ark)
		info, err := iosrc.Stat(ctx, zngpath)
		if err != nil {
			return err
		}
		sum.DataBytes += info.Size()
		sum.RecordCount += int64(chunk.RecordCount)
		if sum.Span.Dur == 0 {
			sum.Span = chunk.Span()
		} else {
			sum.Span = sum.Span.Union(chunk.Span())
		}
		return nil
	})
	return sum, err
}

func (s *Storage) Write(ctx context.Context, zctx *resolver.Context, zr zbuf.Reader) error {
	err := archive.Import(ctx, s.ark, zctx, zr)
	if s.notifier != nil {
		s.notifier.WriterClosed()
	}
	return err
}

type Writer struct {
	*archive.Writer
	notifier WriteNotifier
}

func (w *Writer) Close() error {
	err := w.Writer.Close()
	if w.notifier != nil {
		w.notifier.WriterClosed()
	}
	return err
}

// NewWriter returns an arcive.Writer so data can be written to the archive
// store. When the Writer has been closed a task will be started to compact
// the store.
func (s *Storage) NewWriter(ctx context.Context) *Writer {
	writer := archive.NewWriter(ctx, s.ark)
	return &Writer{writer, s.notifier}
}

func (s *Storage) IndexCreate(ctx context.Context, req api.IndexPostRequest) error {
	var rules []archive.Rule
	if req.AST != nil {
		proc, err := ast.UnpackJSON(nil, req.AST)
		if err != nil {
			return zqe.E(zqe.Invalid, err)
		}
		// XXX IndexPostRequest.Keys hould take a []field.Static or
		// new api.Field type rather than assume embedded "." works
		// as a field separator.  Issue #1463.
		var fields []field.Static
		for _, key := range req.Keys {
			fields = append(fields, field.Dotted(key))
		}
		rule, err := archive.NewRuleAST("zql", proc, req.OutputFile, fields, 0)
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

func (s *Storage) Compact(ctx context.Context) error {
	if err := archive.Compact(ctx, s.ark); err != nil {
		return err
	}
	return archive.Purge(ctx, s.ark)
}
