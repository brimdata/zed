package detector

import (
	"errors"
	"io"
	"net/url"
	"os"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng/resolver"
)

type OpenConfig struct {
	Format         string
	DashStdin      bool
	JSONTypeConfig *ndjsonio.TypeConfig
	JSONPathRegex  string
	AwsCfg         *aws.Config
}

func IsS3Path(path string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	return u.Scheme == "s3"
}

// OpenFile creates and returns zbuf.File for the indicated "path",
// which can be a local file path, a local directory path, or an S3
// URL. If the path is neither of these or can't otherwise be opened,
// an error is returned.
func OpenFile(zctx *resolver.Context, path string, cfg OpenConfig) (*zbuf.File, error) {
	if IsS3Path(path) {
		return OpenS3File(zctx, path, cfg)
	}
	var f *os.File
	if cfg.DashStdin && path == "-" {
		f = os.Stdin
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, errors.New("is a directory")
		}
		f, err = os.Open(path)
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

// OpenS3File opens a file pointed to by an S3-style URL like s3://bucket/name.
//
// The AWS SDK requires the region and credentials (access key ID and
// secret) to make a request to S3. They can be passed as the usual
// AWS environment variables, or be read from the usual aws config
// files in ~/.aws.
//
// Note that access to public objects without credentials is possible
// only if awscfg.AwsCfg.Credentials is set to
// credentials.AnonymousCredentials. However, use of anonymous
// credentials is currently not exposed as a zq command-line option,
// and any attempt to read from S3 without credentials fails.
// (Another way to access such public objects would be through plain
// https access, once we add that support).
func OpenS3File(zctx *resolver.Context, s3path string, cfg OpenConfig) (*zbuf.File, error) {
	u, err := url.Parse(s3path)
	if err != nil {
		return nil, err
	}
	sess := session.Must(session.NewSession(cfg.AwsCfg))
	s3Downloader := s3manager.NewDownloader(sess)
	getObj := &s3.GetObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(u.Path),
	}
	pr, pw := io.Pipe()
	go func() {
		_, err := s3Downloader.Download(&pipeWriterAt{pw}, getObj, func(d *s3manager.Downloader) {
			d.Concurrency = 1
		})
		pw.CloseWithError(err)
	}()
	return OpenFromNamedReadCloser(zctx, pr, s3path, cfg)
}

func OpenFromNamedReadCloser(zctx *resolver.Context, rc io.ReadCloser, path string, cfg OpenConfig) (*zbuf.File, error) {
	var err error
	r := GzipReader(rc)
	var zr zbuf.Reader
	if cfg.Format == "" || cfg.Format == "auto" {
		zr, err = NewReader(r, zctx)
	} else {
		zr, err = LookupReader(r, zctx, cfg.Format)
	}
	if err != nil {
		return nil, err
	}

	if jr, ok := zr.(*ndjsonio.Reader); ok && cfg.JSONTypeConfig != nil {
		if err = jsonConfig(cfg, jr, path); err != nil {
			return nil, err
		}
	}

	return zbuf.NewFile(zr, rc, path), nil
}

func OpenFiles(zctx *resolver.Context, paths ...string) (*zbuf.Combiner, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		reader, err := OpenFile(zctx, path, OpenConfig{})
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	return zbuf.NewCombiner(readers), nil
}

func jsonConfig(cfg OpenConfig, jr *ndjsonio.Reader, filename string) error {
	var path string
	re := regexp.MustCompile(cfg.JSONPathRegex)
	match := re.FindStringSubmatch(filename)
	if len(match) == 2 {
		path = match[1]
	}
	return jr.ConfigureTypes(*cfg.JSONTypeConfig, path)
}
