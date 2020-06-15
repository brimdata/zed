package s3io

import (
	"errors"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/brimsec/zq/pkg/iosource"
)

var DefaultSource = &Source{}
var _ iosource.Source = DefaultSource

type Source struct {
	Config *aws.Config
}

func (s *Source) NewWriter(path string) (io.WriteCloser, error) {
	return NewWriter(path, s.Config)
}

func (s *Source) NewReader(path string) (io.ReadCloser, error) {
	return nil, errors.New("method unsupported")
}
