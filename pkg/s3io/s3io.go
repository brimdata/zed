//go:generate mockgen -destination=mocks/mock_s3.go -package=mocks github.com/aws/aws-sdk-go/service/s3/s3iface S3API

package s3io

import (
	"errors"
	"io"
	"net/url"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var ErrInvalidS3Path = errors.New("path is not a valid s3 location")

func IsS3Path(path string) bool {
	_, _, err := parsePath(path)
	return err == nil
}

func parsePath(path string) (bucket, key string, err error) {
	var u *url.URL
	u, err = url.Parse(path)
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
	uploader *s3manager.Uploader
	bucket   string
	key      string
	once     sync.Once
	done     chan struct{}
	err      error
}

func NewWriter(path string, cfg *aws.Config) (*Writer, error) {
	sess := session.Must(session.NewSession(cfg))
	return NewWriterWithClient(path, s3.New(sess))
}

func NewWriterWithClient(path string, client s3iface.S3API) (*Writer, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	uploader := s3manager.NewUploaderWithClient(client)
	return &Writer{
		bucket:   bucket,
		key:      key,
		uploader: uploader,
		done:     make(chan struct{}),
	}, nil
}

func (w *Writer) init() {
	pr, pw := io.Pipe()
	w.writer = pw
	go func() {
		_, err := w.uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(w.bucket),
			Key:    aws.String(w.key),
			Body:   pr,
		})
		w.err = err
		close(w.done)
		pr.CloseWithError(err)
	}()
}

func (w *Writer) Write(b []byte) (int, error) {
	w.once.Do(w.init)
	select {
	case <-w.done:
		return 0, w.err
	default:
		return w.writer.Write(b)
	}
}

func (w *Writer) Close() error {
	if err := w.writer.Close(); err != nil {
		return err
	}
	<-w.done
	return w.err
}
