package filestore

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqe"
	"github.com/brimsec/zq/zql"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	allZngFile    = "all.zng"
	allZngTmpFile = "all.zng.tmp"
	infoFile      = "info.json"

	defaultStreamSize = 5000
)

var (
	ErrWriteInProgress = errors.New("another write is already in progress")

	zngWriteProc = zql.MustParseProc("sort -r ts")
)

func Load(path string) (*Storage, error) {
	s := &Storage{
		path:       path,
		index:      zngio.NewTimeIndex(),
		streamsize: defaultStreamSize,
		wsem:       semaphore.NewWeighted(1),
	}
	return s, s.readInfoFile()
}

// Storage stores data as a single zng file; this is the default
// storage choice for Brim, where its intended to be a write-once
// import of data.
type Storage struct {
	path       string
	span       nano.Span
	index      *zngio.TimeIndex
	streamsize int
	wsem       *semaphore.Weighted
}

func (s *Storage) NativeDirection() zbuf.Direction {
	return zbuf.DirTimeReverse
}

func (s *Storage) join(args ...string) string {
	args = append([]string{s.path}, args...)
	return filepath.Join(args...)
}

func (s *Storage) Open(_ context.Context, span nano.Span) (zbuf.ReadCloser, error) {
	zctx := resolver.NewContext()
	f, err := os.Open(s.join(allZngFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		// Couldn't read all.zng, check for an old space with all.bzng
		bzngFile := strings.TrimSuffix(allZngFile, filepath.Ext(allZngFile)) + ".bzng"
		f, err = os.Open(s.join(bzngFile))
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			r := zngio.NewReader(strings.NewReader(""), zctx)
			return zbuf.NopReadCloser(r), nil
		}
	}
	return s.index.NewReader(f, zctx, span)
}

type spanWriter struct {
	span nano.Span
}

func (w *spanWriter) Write(rec *zng.Record) error {
	if rec.Ts() == 0 {
		return nil
	}
	first := w.span == nano.Span{}
	s := nano.Span{Ts: rec.Ts(), Dur: 1}
	if first {
		w.span = s
	} else {
		w.span = w.span.Union(s)
	}
	return nil
}

func (s *Storage) Rewrite(ctx context.Context, zr zbuf.Reader) error {
	if !s.wsem.TryAcquire(1) {
		return zqe.E(zqe.Conflict, ErrWriteInProgress)
	}
	defer s.wsem.Release(1)

	// For the time being, this endpoint will overwrite any underlying data.
	// In order to get rid errors on any concurrent searches on this space,
	// write zng to a temp file and rename on successful conversion.
	tmppath := s.join(allZngTmpFile)
	zngfile, err := os.Create(tmppath)
	if err != nil {
		return err
	}

	fileWriter := zngio.NewWriter(zngfile, zio.WriterFlags{StreamRecordsMax: s.streamsize})
	spanWriter := &spanWriter{}
	zw := zbuf.MultiWriter(fileWriter, spanWriter)

	if err := s.write(ctx, zw, zr); err != nil {
		zngfile.Close()
		os.RemoveAll(tmppath)
		return err
	}
	if err := fileWriter.Flush(); err != nil {
		return err
	}

	if err := zngfile.Close(); err != nil {
		os.RemoveAll(tmppath)
		return err
	}
	if err := os.Rename(tmppath, s.join(allZngFile)); err != nil {
		return err
	}
	return s.extendSpan(spanWriter.span)
}

func (s *Storage) write(ctx context.Context, zw zbuf.Writer, zr zbuf.Reader) error {
	out, err := driver.Compile(ctx, zngWriteProc, zr, "ts", false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}
	d := &zngdriver{zw}
	return driver.Run(out, d, nil)
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
	sum.Kind = storage.FileStore
	return sum, nil
}

type info struct {
	MinTime nano.Ts `json:"min_time"`
	MaxTime nano.Ts `json:"max_time"`
}

// readInfoFile reads the info file on disk (if it exists) and sets the cached
// span value for storage.
func (s *Storage) readInfoFile() error {
	b, err := ioutil.ReadFile(s.join(infoFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var inf info
	if err := json.Unmarshal(b, &inf); err != nil {
		return err
	}
	s.span = nano.NewSpanTs(inf.MinTime, inf.MaxTime)
	return nil
}

func (s *Storage) syncInfoFile() error {
	path := s.join(infoFile)
	// If span.Dur is 0 this means we have a zero span and should therefore
	// delete the file.
	if s.span.Dur == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	tmppath := path + ".tmp"
	f, err := fs.Create(tmppath)
	if err != nil {
		return err
	}
	info := info{s.span.Ts, s.span.End()}
	if err := json.NewEncoder(f).Encode(info); err != nil {
		f.Close()
		os.Remove(tmppath)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmppath)
		return err
	}
	return os.Rename(tmppath, path)
}
