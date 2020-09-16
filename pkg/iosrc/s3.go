package iosrc

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
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
var _ ReplacerAble = defaultS3Source

type s3Source struct {
	Config *aws.Config
}

func (s *s3Source) NewWriter(ctx context.Context, u URI) (io.WriteCloser, error) {
	w, err := s3io.NewWriter(ctx, u.String(), s.Config)
	return w, wrapErr(err)
}

func (s *s3Source) NewReader(ctx context.Context, u URI) (Reader, error) {
	r, err := s3io.NewReader(ctx, u.String(), s.Config)
	return r, wrapErr(err)
}

func (s *s3Source) ReadFile(ctx context.Context, u URI) ([]byte, error) {
	r, err := NewReader(ctx, u)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

func (s *s3Source) WriteFile(ctx context.Context, d []byte, u URI) error {
	w, err := NewWriter(ctx, u)
	if err != nil {
		return err
	}
	_, err = w.Write(d)
	if err2 := w.Close(); err == nil {
		err = err2
	}
	return err
}

func (s *s3Source) Remove(ctx context.Context, u URI) error {
	return wrapErr(s3io.Remove(ctx, u.String(), s.Config))
}

func (s *s3Source) RemoveAll(ctx context.Context, u URI) error {
	return wrapErr(s3io.RemoveAll(ctx, u.String(), s.Config))
}

func (s *s3Source) Exists(ctx context.Context, u URI) (bool, error) {
	ok, err := s3io.Exists(ctx, u.String(), s.Config)
	return ok, wrapErr(err)
}

type info s3.HeadObjectOutput

func (i info) Size() int64        { return *i.ContentLength }
func (i info) ModTime() time.Time { return *i.LastModified }

func (s *s3Source) Stat(ctx context.Context, u URI) (Info, error) {
	out, err := s3io.Stat(ctx, u.String(), s.Config)
	if err != nil {
		return nil, wrapErr(err)
	}
	return info(*out), nil
}

func (s *s3Source) NewReplacer(ctx context.Context, uri URI) (io.WriteCloser, error) {
	// Updates to S3 objects are atomic.
	return s.NewWriter(ctx, uri)
}

func wrapErr(err error) error {
	var reqerr awserr.RequestFailure
	if errors.As(err, &reqerr) && reqerr.StatusCode() == http.StatusNotFound {
		return zqe.E(zqe.NotFound)
	}
	return err
}
