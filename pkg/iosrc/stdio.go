package iosrc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

var errStdioNotSupport = errors.New("method not supported with stdio source")

var defaultStdioSource = &StdioSource{}

type StdioSource struct{}

func (f *StdioSource) NewReader(_ context.Context, uri URI) (Reader, error) {
	return getStdioSource(uri)
}

func (s *StdioSource) NewWriter(_ context.Context, uri URI) (io.WriteCloser, error) {
	return getStdioSource(uri)
}

func (s *StdioSource) ReadFile(_ context.Context, uri URI) ([]byte, error) {
	return nil, errStdioNotSupport
}

func (s *StdioSource) WriteFile(_ context.Context, _ []byte, _ URI) error {
	return errStdioNotSupport
}

func (s *StdioSource) WriteFileIfNotExists(_ context.Context, _ []byte, _ URI) error {
	return errStdioNotSupport
}

func (s *StdioSource) Remove(_ context.Context, uri URI) error {
	return errStdioNotSupport
}

func (s *StdioSource) RemoveAll(_ context.Context, uri URI) error {
	return errStdioNotSupport
}

func (s *StdioSource) Stat(_ context.Context, uri URI) (Info, error) {
	return nil, errStdioNotSupport
}

func (s *StdioSource) Exists(_ context.Context, uri URI) (bool, error) {
	if _, err := getStdioSource(uri); err != nil {
		return false, err
	}
	return true, nil
}

func (s *StdioSource) ReadDir(context.Context, URI) ([]Info, error) {
	return nil, errStdioNotSupport
}

func getStdioSource(uri URI) (*os.File, error) {
	if uri.Scheme != "stdio" {
		return nil, fmt.Errorf("scheme of %q must stdio", uri)
	}
	switch uri.Path {
	case "/stdout":
		return os.Stdout, nil
	case "/stdin":
		return os.Stdin, nil
	case "/stderr":
		return os.Stderr, nil
	default:
		return nil, fmt.Errorf("unknown stdio path %q", uri.Path)
	}
}
