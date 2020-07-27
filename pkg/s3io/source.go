package s3io

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqe"
)

var DefaultSource = &Source{}
var _ iosrc.Source = DefaultSource

type Source struct {
	Config *aws.Config
}

func (s *Source) NewWriter(uri iosrc.URI) (io.WriteCloser, error) {
	w, err := NewWriter(uri.String(), s.Config)
	return w, wrapErr(err)
}

func (s *Source) NewReader(uri iosrc.URI) (io.ReadCloser, error) {
	r, err := NewReader(uri.String(), s.Config)
	return r, wrapErr(err)
}

func (s *Source) Remove(uri iosrc.URI) error {
	return wrapErr(Remove(uri.String(), s.Config))
}

func (s *Source) RemoveAll(uri iosrc.URI) error {
	return wrapErr(RemoveAll(uri.String(), s.Config))
}

func (s *Source) Exists(uri iosrc.URI) (bool, error) {
	ok, err := Exists(uri.String(), s.Config)
	return ok, wrapErr(err)
}

type info s3.HeadObjectOutput

func (i info) Size() int64        { return *i.ContentLength }
func (i info) ModTime() time.Time { return *i.LastModified }

func (s *Source) Stat(uri iosrc.URI) (iosrc.Info, error) {
	out, err := Stat(uri.String(), s.Config)
	if err != nil {
		return nil, wrapErr(err)
	}
	return info(*out), nil
}

func (s *Source) NewReplacer(uri iosrc.URI) (io.WriteCloser, error) {
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
