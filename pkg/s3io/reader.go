package s3io

import (
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Reader struct {
	downloader *s3manager.Downloader
	bucket     string
	key        string
	size       int64
	offset     int64
}

func NewReader(path string, cfg *aws.Config) (*Reader, error) {
	info, err := Stat(path, cfg)
	if err != nil {
		return nil, err
	}
	bucket, key, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	client := newClient(cfg)
	downloader := s3manager.NewDownloaderWithClient(client)
	return &Reader{
		downloader: downloader,
		bucket:     bucket,
		key:        key,
		size:       *info.ContentLength,
	}, nil
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += r.offset
	case io.SeekEnd:
		offset += r.size
	default:
		return 0, errors.New("s3io.Reader.Seek: invalid whence")
	}
	if offset < 0 {
		return 0, errors.New("s3io.Reader.Seek: negative position")
	}
	r.offset = offset
	return offset, nil
}

func (r *Reader) bytesRange(num int) string {
	return fmt.Sprintf("bytes=%d-%d", r.offset, r.offset+int64(num)-1)
}

type writeAtBuf []byte

func (w writeAtBuf) WriteAt(p []byte, off int64) (int, error) {
	n := copy(w[off:], p)
	if n < len(p) {
		return n, errors.New("s3io: short write")
	}
	return n, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	getObj := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r.key),
		Range:  aws.String(r.bytesRange(len(p))),
	}
	bytesDownloaded, err := r.downloader.Download(writeAtBuf(p), getObj)
	if err != nil {
		return 0, err
	}

	r.offset += bytesDownloaded

	return int(bytesDownloaded), err
}

func (r *Reader) Close() error {
	return nil
}
