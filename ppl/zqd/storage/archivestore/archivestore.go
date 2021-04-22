package archivestore

import (
	"context"
	"errors"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/immcache"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/ppl/zqd/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"go.uber.org/zap"
)

// XXX this abstraction is upside down.  the API should talk directly to the
// lake and we should get rid of the all.zng file interface.

func Load(ctx context.Context, path iosrc.URI, notifier WriteNotifier, cfg *api.ArchiveConfig, immcache immcache.ImmutableCache) (*Storage, error) {
	// TBD this should be a wrapper around iosrc
	//oo := &lake.OpenOptions{
	//	ImmutableCache: immcache,
	//}

	//XXX we shouldn't do create as a side effect of Load.
	// The client should explicitly create a lake if so desired.
	poolName := "default" //XXX
	lk, err := lake.CreateOrOpen(ctx, path)
	if err != nil {
		return nil, err
	}
	pool, err := lk.OpenPool(ctx, poolName)
	if err != nil {
		//XXX sort keys should be in API
		pool, err = lk.CreatePool(ctx, poolName, field.DottedList("ts"), zbuf.OrderDesc, 0)
		if err != nil {
			return nil, err
		}
	}
	return &Storage{
		pool:     pool,
		notifier: notifier,
	}, nil
}

type WriteNotifier interface {
	WriteNotify()
}

type Storage struct {
	pool     *lake.Pool
	notifier WriteNotifier
}

func NewStorage(pool *lake.Pool) *Storage {
	return &Storage{pool: pool}
}

func (s *Storage) Kind() api.StorageKind {
	return api.ArchiveStore
}

func (s *Storage) NativeOrder() zbuf.Order {
	return s.pool.Order
}

func (s *Storage) MultiSource() driver.MultiSource {
	return lake.NewMultiSourceAt(s.pool, 0)
}

func (s *Storage) StaticSource(src driver.Source) driver.MultiSource {
	return lake.NewStaticSource(s.pool, src)
}

func (s *Storage) Summary(ctx context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = api.ArchiveStore
	head, err := s.pool.Log().Head(ctx)
	if err != nil {
		return sum, err
	}
	ch := make(chan segment.Reference, 10)
	go func() {
		err = s.pool.Scan(ctx, head, ch)
		close(ch)
	}()
	for seg := range ch {
		//XXX should get this from the log
		info, err := iosrc.Stat(ctx, seg.RowObjectPath(s.pool.DataPath))
		if err != nil {
			return sum, err
		}
		sum.DataBytes += info.Size()
		sum.RecordCount += int64(seg.Count)
		if sum.Span.Dur == 0 {
			sum.Span = seg.Span()
		} else {
			sum.Span = sum.Span.Union(seg.Span())
		}
	}
	return sum, err
}

func (s *Storage) Write(ctx context.Context, zctx *zson.Context, zr zbuf.Reader) error {
	commits, err := s.pool.Add(ctx, zr)
	if s.notifier != nil {
		s.notifier.WriteNotify()
	}
	if err == nil {
		err = s.pool.Commit(ctx, commits, 0, "api-user", "<api-blank>")
	}
	return err
}

type Writer struct {
	*lake.Writer
	notifier WriteNotifier
}

func (w *Writer) Close() error {
	err := w.Writer.Close()
	if w.notifier != nil {
		w.notifier.WriteNotify()
	}
	return err
}

// NewWriter returns a writer that will start a compaction when it is closed.
func (s *Storage) NewWriter(ctx context.Context) (*Writer, error) {
	w, err := lake.NewWriter(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	return &Writer{w, s.notifier}, nil
}

func (s *Storage) IndexCreate(ctx context.Context, req api.IndexPostRequest) error {
	var rules []index.Rule
	if req.ZQL != "" {
		// XXX
		// XXX IndexPostRequest.Keys hould take a []field.Static or
		// new api.Field type rather than assume embedded "." works
		// as a field separator.  Issue #1463.
		var fields []field.Static
		for _, key := range req.Keys {
			fields = append(fields, field.Dotted(key))
		}
		rule, err := index.NewZedRule(req.ZQL, req.OutputFile, fields)
		if err != nil {
			return zqe.E(zqe.Invalid, err)
		}
		rules = append(rules, rule)
	}
	for _, pattern := range req.Patterns {
		rule, err := index.ParseRule(pattern)
		if err != nil {
			return zqe.E(zqe.Invalid, err)
		}
		rules = append(rules, rule)
	}
	return errors.New("TBD")
	// XXX Eventually this method should provide progress updates.
	//return lake.ApplyRules(ctx, s.lk, nil, rules...)
}

func (s *Storage) IndexSearch(ctx context.Context, zctx *zson.Context, query index.Query) (zbuf.ReadCloser, error) {
	return nil, errors.New("TBD")
	//return lake.FindReadCloser(ctx, zctx, s.lk, query, lake.AddPath(lake.DefaultAddPathField, false))
}

func (s *Storage) ArchiveStat(ctx context.Context, zctx *zson.Context) (zbuf.ReadCloser, error) {
	return nil, errors.New("TBD")
	//return lake.Stat(ctx, zctx, s.lk)
}

func (s *Storage) Compact(ctx context.Context, logger *zap.Logger) error {
	return errors.New("TBD")
	//return lake.Compact(ctx, s.lk, logger)
}

func (s *Storage) Purge(ctx context.Context, logger *zap.Logger) error {
	return errors.New("TBD")
	//return lake.Purge(ctx, s.lk, logger)
}
