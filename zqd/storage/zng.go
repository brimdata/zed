package storage

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
	"github.com/brimsec/zq/zqe"
	"github.com/brimsec/zq/zql"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

var (
	ErrWriteInProgress = errors.New("another write is already in progress")
)

var zngWriteProc = zql.MustParseProc("sort -r ts")

const (
	allZngFile    = "all.zng"
	allZngTmpFile = "all.zng.tmp"
	infoFile      = "info.json"
)

type ZngStorage struct {
	path       string
	span       nano.Span
	index      *zngio.TimeIndex
	streamsize int
	wsem       *semaphore.Weighted
}

func OpenZng(path string, streamsize int) (*ZngStorage, error) {
	s := &ZngStorage{
		path:       path,
		index:      zngio.NewTimeIndex(),
		streamsize: streamsize,
		wsem:       semaphore.NewWeighted(1),
	}
	return s, s.readInfoFile()
}

func (s *ZngStorage) join(args ...string) string {
	args = append([]string{s.path}, args...)
	return filepath.Join(args...)
}

func (s *ZngStorage) Open(span nano.Span) (zbuf.ReadCloser, error) {
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
	if rec.Ts == 0 {
		return nil
	}
	first := w.span == nano.Span{}
	s := nano.Span{Ts: rec.Ts, Dur: 1}
	if first {
		w.span = s
	} else {
		w.span = w.span.Union(s)
	}
	return nil
}

func (s *ZngStorage) Rewrite(ctx context.Context, zr zbuf.Reader) error {
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
	return s.UnionSpan(spanWriter.span)
}

func (s *ZngStorage) write(ctx context.Context, zw zbuf.Writer, zr zbuf.Reader) error {
	out, err := driver.Compile(ctx, zngWriteProc, zr, false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}
	d := &zngdriver{zw}
	return driver.Run(out, d, nil)
}

// Clear wipes all data from storage. Will wait for any ongoing write operations
// are complete before doing this.
func (s *ZngStorage) Clear(ctx context.Context) error {
	if err := s.wsem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.wsem.Release(1)
	if err := os.Remove(s.join(allZngFile)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return s.UnsetSpan()
}

// XXX This is not thread safe and it should be.
func (s *ZngStorage) UnionSpan(span nano.Span) error {
	first := s.span == nano.Span{}
	if first {
		s.span = span
	} else {
		s.span = s.span.Union(span)
	}
	return s.syncInfoFile()

}

// XXX This is not thread safe and it should be.
func (s *ZngStorage) UnsetSpan() error {
	s.span = nano.Span{}
	return s.syncInfoFile()
}

// XXX This is not thread safe and it should be.
func (s *ZngStorage) Span() nano.Span {
	return s.span
}

func (s *ZngStorage) Size() (int64, error) {
	zngpath := s.join(allZngFile)
	f, err := os.Stat(zngpath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	return f.Size(), nil
}

type info struct {
	MinTime nano.Ts `json:"min_time"`
	MaxTime nano.Ts `json:"max_time"`
}

// readInfoFile reads the info file on disk (if it exists) and sets the cached
// span value for storage.
func (s *ZngStorage) readInfoFile() error {
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

func (s *ZngStorage) syncInfoFile() error {
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
