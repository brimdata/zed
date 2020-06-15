package emitter

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/iosource"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDirS3Source(t *testing.T) {
	path := "s3://testbucket/dir"
	tzng := `
#0:record[_path:string,foo:string]
0:[conn;1;]
#1:record[_path:string,bar:string]
1:[http;2;]`
	mock := &mockLoader{}
	source := &iosource.Registry{}
	source.Add("s3", mock)

	mock.On("NewWriter", "s3://testbucket/dir/conn.tzng").
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)
	mock.On("NewWriter", "s3://testbucket/dir/http.tzng").
		Return(&nopCloser{bytes.NewBuffer(nil)}, nil)

	r := tzngio.NewReader(strings.NewReader(tzng), resolver.NewContext())
	w, err := NewDirWithSource(path, "", os.Stderr, &zio.WriterFlags{Format: "tzng"}, source)
	require.NoError(t, err)
	err = zbuf.Copy(zbuf.NopFlusher(w), r)
	require.NoError(t, err)
	mock.AssertExpectations(t)
}

func TestDirUnknownSource(t *testing.T) {
	source := &iosource.Registry{}
	path := "unknown://path/unknown"
	_, err := NewDirWithSource(path, "", os.Stderr, &zio.WriterFlags{Format: "tzng"}, source)
	require.EqualError(t, err, "unknown: unsupported scheme")
}

type nopCloser struct{ *bytes.Buffer }

func (nopCloser) Close() error { return nil }

type mockLoader struct {
	mock.Mock
}

func (s *mockLoader) NewReader(path string) (io.ReadCloser, error) {
	args := s.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (s *mockLoader) NewWriter(path string) (io.WriteCloser, error) {
	args := s.Called(path)
	return args.Get(0).(io.WriteCloser), args.Error(1)
}
