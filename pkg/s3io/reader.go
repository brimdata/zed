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

func (r *Reader) Read(p []byte) (int, error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}

	n := len(p)
	getObjRange := r.bytesRange(n)
	getObj := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(r.key),
	}
	if len(getObjRange) > 0 {
		getObj.Range = aws.String(getObjRange)
	}

	wab := aws.NewWriteAtBuffer(p)
	bytesDownloaded, err := r.downloader.Download(wab, getObj)
	if err != nil {
		return 0, err
	}

	buf := wab.Bytes()
	if len(buf) > n {
		// backing buffer reassigned, copy over some of the data
		copy(p, buf)
		bytesDownloaded = int64(n)
	}
	r.offset += bytesDownloaded

	return int(bytesDownloaded), err
}

func (r *Reader) Close() error {
	return nil
}
