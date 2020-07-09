package detector

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/s3io"
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

	if s3io.IsS3Path(path) {
		f, err := s3io.NewReader(path, cfg.AwsCfg)
		if err != nil {
			return nil, err
		}
		return OpenFromNamedReadCloser(zctx, f, path, cfg)
	}

	var f *os.File
	if path == StdinPath {
		f = os.Stdin
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, errors.New("is a directory")
		}
		f, err = fs.Open(path)
		if err != nil {
			return nil, err
		}
	}

	return OpenFromNamedReadCloser(zctx, f, path, cfg)
}

type pipeWriterAt struct {
	*io.PipeWriter
}

func (pw *pipeWriterAt) WriteAt(p []byte, _ int64) (n int, err error) {
	return pw.Write(p)
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

func OpenFiles(zctx *resolver.Context, dir zbuf.RecordCmpFn, paths ...string) (*zbuf.Combiner, error) {
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

func (r *multiFileReader) Read() (rec *zng.Record, err error) {
again:
	if r.reader == nil {
		if len(r.paths) == 0 {
			return nil, nil
		}
		path := r.paths[0]
		r.paths = r.paths[1:]
		r.reader, err = OpenFile(r.zctx, path, r.cfg)
		if err != nil {
			return nil, err
		}
	}
	rec, err = r.reader.Read()
	if err == nil && rec == nil {
		r.reader.Close()
		r.reader = nil
		goto again
	}
	return
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
