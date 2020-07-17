package s3io

import (
	"errors"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/brimsec/zq/pkg/iosrc"
)

var ErrInvalidS3Path = errors.New("path is not a valid s3 location")

// uploader is an interface wrapper for s3manager.Uploader. This is only here
// for unit testing purposes.
type uploader interface {
	Upload(*s3manager.UploadInput, ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)
}

func IsS3Path(path string) bool {
	_, _, err := parsePath(path)
	return err == nil
}

func parsePath(path string) (bucket, key string, err error) {
	u, err := iosrc.ParseURI(path)
	if err != nil {
		return
	}
	if u.Scheme != "s3" {
		err = ErrInvalidS3Path
	}
	bucket = u.Host
	key = u.Path
	return
}

type Writer struct {
	writer   *io.PipeWriter
	reader   *io.PipeReader
	uploader uploader
	bucket   string
	key      string
	once     sync.Once
	done     sync.WaitGroup
	err      error
}

func NewWriter(path string, cfg *aws.Config, options ...func(*s3manager.Uploader)) (*Writer, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	client := newClient(cfg)
	uploader := s3manager.NewUploaderWithClient(client, options...)
	pr, pw := io.Pipe()
	return &Writer{
		bucket:   bucket,
		key:      key,
		writer:   pw,
		reader:   pr,
		uploader: uploader,
	}, nil
}

func (w *Writer) init() {
	w.done.Add(1)
	go func() {
		_, err := w.uploader.Upload(&s3manager.UploadInput{
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

func Stat(path string, cfg *aws.Config) (*s3.HeadObjectOutput, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	return newClient(cfg).HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
}

func ListObjects(path string, cfg *aws.Config) ([]string, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	client := newClient(cfg)
	var keys []string
	return keys, ls(client, bucket, key, func(out *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range out.Contents {
			keys = append(keys, *obj.Key)
		}
		return true
	})
}

func ls(client *s3.S3, bucket, key string, cb func(*s3.ListObjectsV2Output, bool) bool) error {
	input := &s3.ListObjectsV2Input{
		Prefix: aws.String(key),
		Bucket: aws.String(bucket),
	}
	return client.ListObjectsV2Pages(input, cb)
}

func Exists(path string, cfg *aws.Config) (bool, error) {
	_, err := Stat(path, cfg)
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
	sess := session.Must(session.NewSession(cfg))
	return s3.New(sess)
}
