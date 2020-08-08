package iosrc

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/brimsec/zq/pkg/s3io"
	"github.com/brimsec/zq/zqe"
)

var defaultS3Source = &s3Source{}
var _ Source = defaultS3Source

type s3Source struct {
	Config *aws.Config
}

func (s *s3Source) NewWriter(u URI) (io.WriteCloser, error) {
	w, err := s3io.NewWriter(u.String(), s.Config)
	return w, wrapErr(err)
}

func (s *s3Source) NewReader(u URI) (Reader, error) {
	r, err := s3io.NewReader(u.String(), s.Config)
	return r, wrapErr(err)
}

func (s *s3Source) Remove(u URI) error {
	return wrapErr(s3io.Remove(u.String(), s.Config))
}

func (s *s3Source) RemoveAll(u URI) error {
	return wrapErr(s3io.RemoveAll(u.String(), s.Config))
}

func (s *s3Source) Exists(u URI) (bool, error) {
	ok, err := s3io.Exists(u.String(), s.Config)
	return ok, wrapErr(err)
}

type info s3.HeadObjectOutput

func (i info) Size() int64        { return *i.ContentLength }
func (i info) ModTime() time.Time { return *i.LastModified }

func (s *s3Source) Stat(u URI) (Info, error) {
	out, err := s3io.Stat(u.String(), s.Config)
	if err != nil {
		return nil, wrapErr(err)
	}
	return info(*out), nil
}

func (s *s3Source) NewReplacer(uri URI) (io.WriteCloser, error) {
	// Updates to S3 objects are atomic.
	return s.NewWriter(uri)
}

func wrapErr(err error) error {
	var reqerr awserr.RequestFailure
	if errors.As(err, &reqerr) && reqerr.StatusCode() == http.StatusNotFound {
		return zqe.E(zqe.NotFound)
	}
	return err
}
