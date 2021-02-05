package filestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/compiler"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/ppl/zqd/storage"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/azngio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	allZngFile = "all.zng"
	infoFile   = "info.json"
)

var (
	ErrWriteInProgress = errors.New("another write is already in progress")

	zngWriteProc = compiler.MustParseProc("sort -r ts")
)

func Load(path iosrc.URI, logger *zap.Logger) (*Storage, error) {
	if path.Scheme != "file" {
		return nil, fmt.Errorf("unsupported FileStore scheme %q", path)
	}
	s := &Storage{
		index:      zngio.NewTimeIndex(),
		logger:     logger,
		path:       path.Filepath(),
		streamsize: zngio.DefaultStreamRecordsMax,
		wsem:       semaphore.NewWeighted(1),
	}
	return s, s.readInfoFile()
}

// Storage stores data as a single zng file; this is the default
// storage choice for Brim, where its intended to be a write-once
// import of data.
type Storage struct {
	alphaMigrated bool
	index         *zngio.TimeIndex
	logger        *zap.Logger
	path          string
	span          nano.Span
	streamsize    int
	wsem          *semaphore.Weighted
}

func (s *Storage) Kind() api.StorageKind {
	return api.FileStore
}

func (s *Storage) NativeOrder() zbuf.Order {
	return zbuf.OrderDesc
}

func (s *Storage) join(args ...string) string {
	args = append([]string{s.path}, args...)
	return filepath.Join(args...)
}

func (s *Storage) Open(ctx context.Context, zctx *resolver.Context, span nano.Span) (zbuf.ReadCloser, error) {
	if err := s.migrateAlphaZngIfNeeded(ctx); err != nil {
		return nil, err
	}
	f, err := fs.Open(s.join(allZngFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		r := zngio.NewReader(strings.NewReader(""), zctx)
		return zbuf.NopReadCloser(r), nil
	}
	return s.index.NewReader(f, zctx, span)
}

type spanWriter struct {
	span   nano.Span
	writes bool
}

func (w *spanWriter) Write(rec *zng.Record) error {
	if rec.Ts() == 0 {
		return nil
	}
	w.writes = true
	first := w.span == nano.Span{}
	s := nano.Span{Ts: rec.Ts(), Dur: 1}
	if first {
		w.span = s
	} else {
		w.span = w.span.Union(s)
	}
	return nil
}

func (s *Storage) Write(ctx context.Context, zctx *resolver.Context, zr zbuf.Reader) error {
	if !s.wsem.TryAcquire(1) {
		return zqe.E(zqe.Conflict, ErrWriteInProgress)
	}
	defer s.wsem.Release(1)

	spanWriter := &spanWriter{}
	if err := fs.ReplaceFile(s.join(allZngFile), 0600, func(w io.Writer) error {
		fileWriter := zngio.NewWriter(bufwriter.New(zio.NopCloser(w)), zngio.WriterOpts{
			StreamRecordsMax: s.streamsize,
			LZ4BlockSize:     zngio.DefaultLZ4BlockSize,
		})
		zw := zbuf.MultiWriter(fileWriter, spanWriter)
		if err := driver.Copy(ctx, zw, zngWriteProc, zctx, zr, driver.Config{}); err != nil {
			return err
		}
		return fileWriter.Close()
	}); err != nil {
		return err
	}

	if !spanWriter.writes {
		return nil
	}

	return s.extendSpan(spanWriter.span)
}

// Clear wipes all data from storage. Will wait for any ongoing write operations
// are complete before doing this.
func (s *Storage) Clear(ctx context.Context) error {
	if err := s.wsem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.wsem.Release(1)
	if err := os.Remove(s.join(allZngFile)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.SetSpan(nano.Span{})
}

func (s *Storage) extendSpan(span nano.Span) error {
	// XXX This is not thread safe and it should be.
	first := s.span == nano.Span{}
	if first {
		s.span = span
	} else {
		s.span = s.span.Union(span)
	}
	return s.syncInfoFile()

}

func (s *Storage) SetSpan(span nano.Span) error {
	// XXX This is not thread safe and it should be.
	s.span = span
	return s.syncInfoFile()
}

func (s *Storage) Summary(_ context.Context) (storage.Summary, error) {
	var sum storage.Summary
	zngpath := s.join(allZngFile)
	if f, err := os.Stat(zngpath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return sum, err
		}
	} else {
		sum.DataBytes = f.Size()
	}
	// XXX This is not thread safe and it should be.
	sum.Span = s.span
	sum.Kind = api.FileStore
	return sum, nil
}

type info struct {
	AlphaMigrated bool    `json:"alpha_migrated"`
	MinTime       nano.Ts `json:"min_time"`
	MaxTime       nano.Ts `json:"max_time"`
}

// readInfoFile reads the info file on disk (if it exists) and sets the cached
// span value for storage.
func (s *Storage) readInfoFile() error {
	var inf info
	if err := fs.UnmarshalJSONFile(s.join(infoFile), &inf); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		inf.AlphaMigrated = true
	}
	s.span = nano.NewSpanTs(inf.MinTime, inf.MaxTime)
	s.alphaMigrated = inf.AlphaMigrated
	return nil
}

func (s *Storage) syncInfoFile() error {
	// If we are updating the info file, we've either written data, or are clearing
	// the info file so that we can write data later. In either case, we'd be using
	// the post-alpha zng format.
	s.alphaMigrated = true

	path := s.join(infoFile)
	// If span.Dur is 0 this means we have a zero span and should therefore
	// delete the file.
	if s.span.Dur == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	info := info{
		MinTime:       s.span.Ts,
		MaxTime:       s.span.End(),
		AlphaMigrated: s.alphaMigrated,
	}
	return fs.MarshalJSONFile(info, path, 0600)
}

func (s *Storage) migrateAlphaZngIfNeeded(ctx context.Context) error {
	if s.alphaMigrated {
		return nil
	}
	if err := s.wsem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.wsem.Release(1)
	exists, err := s.ensureAllZngFile()
	if err != nil {
		return err
	}
	if !exists {
		s.alphaMigrated = true
		return nil
	}
	isAlpha, err := isAlphaZngFile(s.join(allZngFile))
	if err != nil {
		return err
	}
	if isAlpha {
		if err := s.migrateAlphaZngFile(); err != nil {
			return err
		}
	}
	return s.syncInfoFile()
}

// ensureAllZngFile returns true if there is a data file for this file store.
// If an older file name of 'all.bzng' is found, it is renamed to all.zng.
func (s *Storage) ensureAllZngFile() (bool, error) {
	_, err := os.Stat(s.join(allZngFile))
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err == nil {
		return true, nil
	}
	allBzng := s.join("all.bzng")
	_, err = os.Stat(allBzng)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, os.Rename(allBzng, s.join(allZngFile))
}

func isAlphaZngFile(fpath string) (bool, error) {
	zfile, err := detector.OpenFile(resolver.NewContext(), fpath, zio.ReaderOpts{})
	if err != nil {
		return false, err
	}
	defer zfile.Close()
	if _, ok := zfile.Reader.(*azngio.Reader); ok {
		return true, nil
	}
	return false, nil
}

// migrateAlphaZngFile uses the azngio package to read the alpha zng file
// for this file store, and replaces it with a zng file using the post-alpha
// format.
func (s *Storage) migrateAlphaZngFile() (err error) {
	fpath := s.join(allZngFile)
	s.logger.Info("Alpha zng migration start", zap.String("file", fpath))
	defer func() {
		if err != nil {
			s.logger.Info("Alpha zng migration failure", zap.String("file", fpath), zap.Error(err))
			return
		}
		s.logger.Info("Alpha zng migration success", zap.String("file", fpath))
	}()
	return fs.ReplaceFile(fpath, 0666, func(w io.Writer) error {
		zctx := resolver.NewContext()
		zw := zngio.NewWriter(bufwriter.New(zio.NopCloser(w)), zngio.WriterOpts{
			StreamRecordsMax: zngio.DefaultStreamRecordsMax,
			LZ4BlockSize:     zngio.DefaultLZ4BlockSize,
		})
		rc, err := fs.Open(fpath)
		if err != nil {
			return err
		}
		ar, err := azngio.NewReader(rc, zctx)
		if err != nil {
			rc.Close()
			return err
		}
		err = zbuf.Copy(zw, ar)
		if rcErr := rc.Close(); err == nil {
			err = rcErr
		}
		if zwErr := zw.Close(); err == nil {
			err = zwErr
		}
		return err
	})
}

type Migrator struct {
	ctx context.Context
	sem *semaphore.Weighted
}

func NewMigrator(ctx context.Context) *Migrator {
	return &Migrator{
		ctx: ctx,
		sem: semaphore.NewWeighted(int64(runtime.GOMAXPROCS(0))),
	}
}

func (m *Migrator) Add(s *Storage) {
	if s.alphaMigrated {
		return
	}
	go func() {
		if err := m.sem.Acquire(m.ctx, 1); err != nil {
			return
		}
		defer m.sem.Release(1)
		// Error handling and logging handled inside the migrate call.
		_ = s.migrateAlphaZngIfNeeded(m.ctx)
	}()
}
