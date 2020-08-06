package iosrc

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var errStdioNotSupport = errors.New("method not supported with stdio source")

var defaultStdioSource = &StdioSource{}

type StdioSource struct{}

func (f *StdioSource) NewReader(uri URI) (Reader, error) {
	return getStdioSource(uri)
}

func (s *StdioSource) NewWriter(uri URI) (io.WriteCloser, error) {
	return getStdioSource(uri)
}

func (s *StdioSource) Remove(uri URI) error {
	return errStdioNotSupport
}

func (s *StdioSource) RemoveAll(uri URI) error {
	return errStdioNotSupport
}

func (s *StdioSource) Stat(uri URI) (Info, error) {
	return nil, errStdioNotSupport
}

func (s *StdioSource) Exists(uri URI) (bool, error) {
	if _, err := getStdioSource(uri); err != nil {
		return false, err
	}
	return true, nil
}

func getStdioSource(uri URI) (*os.File, error) {
	if uri.Scheme != "stdio" {
		return nil, fmt.Errorf("scheme of %q must stdio", uri)
	}
	switch uri.Path {
	default:
		return nil, fmt.Errorf("unknown stdio path %q", uri.Path)
	case "/stdout":
		return os.Stdout, nil
	case "/stdin":
		return os.Stdin, nil
	case "/stderr":
		return os.Stderr, nil
	}

}
