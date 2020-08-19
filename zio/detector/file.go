package detector

import (
	"context"
	"io"
	"net/url"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/s3io"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zio/parquetio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"

	"github.com/xitongsys/parquet-go-source/local"
	parquets3 "github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/source"
)

type OpenConfig struct {
	Format         string
	JSONTypeConfig *ndjsonio.TypeConfig
	JSONPathRegex  string
	AwsCfg         *aws.Config
}

const StdinPath = "/dev/stdin"

// OpenFile creates and returns zbuf.File for the indicated "path",
// which can be a local file path, a local directory path, or an S3
// URL. If the path is neither of these or can't otherwise be opened,
// an error is returned.
func OpenFile(zctx *resolver.Context, path string, cfg OpenConfig) (*zbuf.File, error) {
	// Parquet is special and needs its own reader for s3 sources- therefore this must go before
	// the IsS3Path check.
	if cfg.Format == "parquet" {
		return OpenParquet(zctx, path, cfg)
	}

	var f io.ReadCloser
	if path == StdinPath {
		f = os.Stdin
	} else {
		uri, err := iosrc.ParseURI(path)
		if err != nil {
			return nil, err
		}
		if f, err = iosrc.NewReader(uri); err != nil {
			return nil, err
		}
	}

	return OpenFromNamedReadCloser(zctx, f, path, cfg)
}

func OpenParquet(zctx *resolver.Context, path string, cfg OpenConfig) (*zbuf.File, error) {
	var pf source.ParquetFile
	var err error
	if s3io.IsS3Path(path) {
		var u *url.URL
		u, err = url.Parse(path)
		if err != nil {
			return nil, err
		}
		pf, err = parquets3.NewS3FileReader(context.Background(), u.Host, u.Path, cfg.AwsCfg)
	} else {
		pf, err = local.NewLocalFileReader(path)
	}
	if err != nil {
		return nil, err
	}

	r, err := parquetio.NewReader(pf, zctx, parquetio.ReaderOpts{})
	if err != nil {
		return nil, err
	}
	return zbuf.NewFile(r, pf, path), nil
}

func OpenFromNamedReadCloser(zctx *resolver.Context, rc io.ReadCloser, path string, cfg OpenConfig) (*zbuf.File, error) {
	var err error
	r := GzipReader(rc)
	var zr zbuf.Reader
	if cfg.Format == "" || cfg.Format == "auto" {
		zr, err = NewReaderWithConfig(r, zctx, path, cfg)
	} else {
		zr, err = lookupReader(r, zctx, path, cfg)
	}
	if err != nil {
		return nil, err
	}

	return zbuf.NewFile(zr, rc, path), nil
}

func OpenFiles(zctx *resolver.Context, dir zbuf.RecordCmpFn, paths ...string) (zbuf.ReadCloser, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		reader, err := OpenFile(zctx, path, OpenConfig{})
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	return zbuf.NewCombiner(readers, dir), nil
}

type multiFileReader struct {
	reader *zbuf.File
	zctx   *resolver.Context
	paths  []string
	cfg    OpenConfig
}

var _ zbuf.ReadCloser = (*multiFileReader)(nil)
var _ scanner.ScannerAble = (*multiFileReader)(nil)

// MultiFileReader returns a zbuf.ReadCloser that's the logical concatenation
// of the provided input paths. They're read sequentially. Once all inputs have
// reached end of stream, Read will return end of stream. If any of the readers
// return a non-nil error, Read will return that error.
func MultiFileReader(zctx *resolver.Context, paths []string, cfg OpenConfig) zbuf.ReadCloser {
	return &multiFileReader{
		zctx:  zctx,
		paths: paths,
		cfg:   cfg,
	}
}

func (r *multiFileReader) prepReader() (bool, error) {
	if r.reader == nil {
		if len(r.paths) == 0 {
			return true, nil
		}
		path := r.paths[0]
		r.paths = r.paths[1:]
		rdr, err := OpenFile(r.zctx, path, r.cfg)
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

func (r *multiFileReader) NewScanner(ctx context.Context, f filter.Filter, filterExpr ast.BooleanExpr, s nano.Span) (scanner.Scanner, error) {
	return &multiFileScanner{
		multiFileReader: r,
		ctx:             ctx,
		filter:          f,
		filterExpr:      filterExpr,
		span:            s,
	}, nil
}

type multiFileScanner struct {
	*multiFileReader
	ctx        context.Context
	filter     filter.Filter
	filterExpr ast.BooleanExpr
	span       nano.Span

	mu      sync.Mutex // protects below
	scanner scanner.Scanner
	stats   scanner.ScannerStats
}

func (s *multiFileScanner) Pull() (zbuf.Batch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if done, err := s.prepReader(); done || err != nil {
			return nil, err
		}
		if s.scanner == nil {
			sn, err := scanner.NewScanner(s.ctx, s.reader, s.filter, s.filterExpr, s.span)
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

func (s *multiFileScanner) Stats() *scanner.ScannerStats {
	s.mu.Lock()
	st := s.stats
	if s.scanner != nil {
		st.Accumulate(s.scanner.Stats())
	}
	s.mu.Unlock()
	return &st
}
