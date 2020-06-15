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

func (l *Source) NewWriter(path string) (io.WriteCloser, error) {
	return NewWriter(path, l.Config)
}

func (l *Source) NewReader(path string) (io.ReadCloser, error) {
	return nil, errors.New("method unsupported")
}
