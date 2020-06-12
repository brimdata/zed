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
	sess := session.Must(session.NewSession(cfg))
	uploader := s3manager.NewUploader(sess, options...)
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
