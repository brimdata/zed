package detector

import (
	"context"
	"io"
	"sync"

	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zio"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

// OpenFile creates and returns zbuf.File for the indicated "path",
// which can be a local file path, a local directory path, or an S3
// URL. If the path is neither of these or can't otherwise be opened,
// an error is returned.
func OpenFile(zctx *resolver.Context, path string, opts zio.ReaderOpts) (*zbuf.File, error) {
	return OpenFileWithContext(context.Background(), zctx, path, opts)
}

func OpenFileWithContext(ctx context.Context, zctx *resolver.Context, path string, opts zio.ReaderOpts) (*zbuf.File, error) {
	uri, err := iosrc.ParseURI(path)
	if err != nil {
		return nil, err
	}
	f, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return nil, err
	}
	return OpenFromNamedReadCloser(zctx, f, path, opts)
}

func OpenFromNamedReadCloser(zctx *resolver.Context, rc io.ReadCloser, path string, opts zio.ReaderOpts) (*zbuf.File, error) {
	var err error
	r := io.Reader(rc)
	if opts.Format != "parquet" && opts.Format != "zst" {
		r = GzipReader(rc)
	}
	var zr zbuf.Reader
	if opts.Format == "" || opts.Format == "auto" {
		zr, err = NewReaderWithOpts(r, zctx, path, opts)
	} else {
		zr, err = lookupReader(r, zctx, path, opts)
	}
	if err != nil {
		return nil, err
	}

	return zbuf.NewFile(zr, rc, path), nil
}

func OpenFiles(ctx context.Context, zctx *resolver.Context, paths ...string) ([]zbuf.Reader, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		reader, err := OpenFileWithContext(ctx, zctx, path, zio.ReaderOpts{})
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	return readers, nil
}

type multiFileReader struct {
	reader *zbuf.File
	ctx    context.Context
	zctx   *resolver.Context
	paths  []string
	opts   zio.ReaderOpts
}

var _ zbuf.ReadCloser = (*multiFileReader)(nil)
var _ zbuf.ScannerAble = (*multiFileReader)(nil)

// MultiFileReader returns a zbuf.ReadCloser that's the logical concatenation
// of the provided input paths. They're read sequentially. Once all inputs have
// reached end of stream, Read will return end of stream. If any of the readers
// return a non-nil error, Read will return that error.
func MultiFileReader(zctx *resolver.Context, paths []string, opts zio.ReaderOpts) zbuf.ReadCloser {
	return MultiFileReaderWithContext(context.Background(), zctx, paths, opts)
}

func MultiFileReaderWithContext(ctx context.Context, zctx *resolver.Context, paths []string, opts zio.ReaderOpts) zbuf.ReadCloser {
	return &multiFileReader{
		ctx:   ctx,
		zctx:  zctx,
		paths: paths,
		opts:  opts,
	}
}

func (r *multiFileReader) prepReader() (bool, error) {
	if r.reader == nil {
		if len(r.paths) == 0 {
			return true, nil
		}
		path := r.paths[0]
		r.paths = r.paths[1:]
		rdr, err := OpenFileWithContext(r.ctx, r.zctx, path, r.opts)
		if err != nil {
			return false, err
		}
		r.reader = rdr
	}
	return false, nil
}

func (r *multiFileReader) Read() (*zng.Record, error) {
	for {
		if done, err := r.prepReader(); done || err != nil {
			return nil, err
		}
		rec, err := r.reader.Read()
		if err == nil && rec == nil {
			r.reader.Close()
			r.reader = nil
			continue
		}
		return rec, err
	}
}

// Close closes the current open files and clears the list of remaining paths
// to be read. This is not thread safe.
func (r *multiFileReader) Close() (err error) {
	if r.reader != nil {
		err = r.reader.Close()
		r.reader = nil
	}
	return
}

func (r *multiFileReader) NewScanner(ctx context.Context, filter zbuf.Filter, s nano.Span) (zbuf.Scanner, error) {
	return &multiFileScanner{
		multiFileReader: r,
		ctx:             ctx,
		filter:          filter,
		span:            s,
	}, nil
}

type multiFileScanner struct {
	*multiFileReader
	ctx    context.Context
	filter zbuf.Filter
	span   nano.Span

	mu      sync.Mutex // protects below
	scanner zbuf.Scanner
	stats   zbuf.ScannerStats
}

func (s *multiFileScanner) Pull() (zbuf.Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if done, err := s.prepReader(); done || err != nil {
			return nil, err
		}
		if s.scanner == nil {
			sn, err := zbuf.NewScanner(s.ctx, s.reader, s.filter, s.span)
			if err != nil {
				return nil, err
			}
			s.scanner = sn
		}
		batch, err := s.scanner.Pull()
		if err == nil && batch == nil {
			s.stats.Accumulate(s.scanner.Stats())
			s.scanner = nil
			s.reader.Close()
			s.reader = nil
			continue
		}
		return batch, err
	}
}

func (s *multiFileScanner) Stats() *zbuf.ScannerStats {
	s.mu.Lock()
	st := s.stats
	if s.scanner != nil {
		st.Accumulate(s.scanner.Stats())
	}
	s.mu.Unlock()
	return &st
}
