package s3io

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var ErrInvalidS3Path = errors.New("path is not a valid s3 location")

// uploader is an interface wrapper for s3manager.Uploader. This is only here
// for unit testing purposes.
type uploader interface {
	UploadWithContext(context.Context, *s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

func IsS3Path(path string) bool {
	_, _, err := parsePath(path)
	return err == nil
}

func parsePath(path string) (bucket, key string, err error) {
	u, err := url.Parse(path)
	if err != nil {
		return
	}
	if u.Scheme != "s3" {
		err = ErrInvalidS3Path
	}
	bucket = u.Host
	key = strings.TrimPrefix(u.Path, "/")
	return
}

type Writer struct {
	writer   *io.PipeWriter
	reader   *io.PipeReader
	ctx      context.Context
	uploader uploader
	bucket   string
	key      string
	once     sync.Once
	done     sync.WaitGroup
	err      error
}

func NewWriter(ctx context.Context, path string, cfg *aws.Config, options ...func(*s3manager.Uploader)) (*Writer, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	client := newClient(cfg)
	uploader := s3manager.NewUploaderWithClient(client, options...)
	pr, pw := io.Pipe()
	return &Writer{
		writer:   pw,
		reader:   pr,
		ctx:      ctx,
		uploader: uploader,
		bucket:   bucket,
		key:      key,
	}, nil
}

func (w *Writer) init() {
	w.done.Add(1)
	go func() {
		_, err := w.uploader.UploadWithContext(w.ctx, &s3manager.UploadInput{
			Bucket: aws.String(w.bucket),
			Key:    aws.String(w.key),
			Body:   w.reader,
		})
		w.err = err
		_ = w.reader.CloseWithError(err) // can ignore, return value will always be nil
		w.done.Done()
	}()
}

func (w *Writer) Write(b []byte) (int, error) {
	w.once.Do(w.init)
	return w.writer.Write(b)
}

func (w *Writer) Close() error {
	err := w.writer.Close()
	w.done.Wait()
	if err != nil {
		return err
	}
	return w.err
}

func RemoveAll(ctx context.Context, path string, cfg *aws.Config) error {
	bucket, key, err := parsePath(path)
	if err != nil {
		return err
	}
	client := newClient(cfg)
	deleter := s3manager.NewBatchDeleteWithClient(client)
	it := s3manager.NewDeleteListIterator(client, &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(key),
	})
	if err := deleter.Delete(ctx, it); err != nil {
		return err
	}
	for it.Next() {
		it.DeleteObject()
		if err := it.Err(); err != nil {
			return err
		}
	}
	return nil
}

func Remove(ctx context.Context, path string, cfg *aws.Config) error {
	bucket, key, err := parsePath(path)
	if err != nil {
		return err
	}
	if _, err := head(ctx, bucket, key, cfg); err != nil {
		return err
	}
	_, err = newClient(cfg).DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Key:    &key,
		Bucket: &bucket,
	})
	return err
}

type Info struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

func Stat(ctx context.Context, uri string, cfg *aws.Config) (Info, error) {
	bucket, key, err := parsePath(uri)
	if err != nil {
		return Info{}, err
	}
	h, err := head(ctx, bucket, key, cfg)
	if err != nil {
		return Info{}, err
	}
	return Info{
		Name:    path.Base(key),
		Size:    *h.ContentLength,
		ModTime: *h.LastModified,
		IsDir:   false,
	}, nil
}

func head(ctx context.Context, bucket, key string, cfg *aws.Config) (*s3.HeadObjectOutput, error) {
	return newClient(cfg).HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

func List(ctx context.Context, path string, cfg *aws.Config) ([]Info, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	client := newClient(cfg)
	input := &s3.ListObjectsV2Input{
		Prefix:    aws.String(key),
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
	}
	var entries []Info
	err = client.ListObjectsV2PagesWithContext(ctx, input, func(out *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range out.Contents {
			entries = append(entries, Info{
				Name:    strings.TrimPrefix(*obj.Key, key),
				Size:    *obj.Size,
				ModTime: *obj.LastModified,
				IsDir:   false,
			})
		}
		for _, p := range out.CommonPrefixes {
			entries = append(entries, Info{
				Name:  strings.TrimSuffix(strings.TrimPrefix(*p.Prefix, key), "/"),
				IsDir: true,
			})
		}
		return true
	})
	return entries, err
}

func Exists(ctx context.Context, path string, cfg *aws.Config) (bool, error) {
	_, err := Stat(ctx, path, cfg)
	if err != nil {
		var reqerr awserr.RequestFailure
		if errors.As(err, &reqerr) && reqerr.StatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func newClient(cfg *aws.Config) *s3.S3 {
	if cfg == nil {
		cfg = &aws.Config{}
	}
	// Add ability to override s3 endpoint via env variable (the aws sdk doesn't
	// support this). This is mostly for system tests w/ minio.
	if endpoint := os.Getenv("AWS_S3_ENDPOINT"); cfg.Endpoint == nil && endpoint != "" {
		cfg.Endpoint = aws.String(endpoint)
		cfg.S3ForcePathStyle = aws.Bool(true) // https://github.com/minio/minio/tree/master/docs/config#domain
	}

	// Unless the user has a environment setting for shared config, enable it
	// so that region & other info is automatically picked up from the
	// .aws/config file.
	scs := session.SharedConfigEnable
	if os.Getenv("AWS_SDK_LOAD_CONFIG") != "" {
		scs = session.SharedConfigStateFromEnv
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config:            *cfg,
		SharedConfigState: scs,
	}))
	return s3.New(sess)
}
