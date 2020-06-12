package s3io

import (
	"errors"
	"io"
	"net/url"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	uploader uploader
	bucket   string
	key      string
	once     sync.Once
	done     chan struct{}
	err      error
}

func NewWriter(path string, cfg *aws.Config, options ...func(*s3manager.Uploader)) (*Writer, error) {
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	sess := session.Must(session.NewSession(cfg))
	uploader := s3manager.NewUploader(sess, options...)
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
		_ = pr.CloseWithError(err) // can ignore, return value will always be nil
	}()
}

func (w *Writer) Write(b []byte) (int, error) {
	w.once.Do(w.init)
	return w.writer.Write(b)
}

func (w *Writer) Close() error {
	err := w.writer.Close()
	<-w.done
	if err != nil {
		return err
	}
	return w.err
}
