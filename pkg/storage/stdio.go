package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

var errStdioNotSupport = errors.New("method not supported with stdio source")

type StdioEngine struct{}

func NewStdioEngine() *StdioEngine {
	return &StdioEngine{}
}

func (*StdioEngine) Get(_ context.Context, u *URI) (Reader, error) {
	if u.Scheme != "stdio" || (u.Path != "stdin" && u.Path != "") {
		return nil, fmt.Errorf("cannot read from %q", u)
	}
	return &nopReadAtCloser{os.Stdin}, nil
}

func (*StdioEngine) Put(ctx context.Context, u *URI) (io.WriteCloser, error) {
	var f *os.File
	switch u.Path {
	case "stdout", "":
		f = os.Stdout
	case "stderr":
		f = os.Stderr
	default:
		return nil, fmt.Errorf("cannot write to '%s'", u.Path)
	}
	return &NopCloser{f}, nil
}

func (*StdioEngine) PutIfNotExists(context.Context, *URI, []byte) error {
	return errStdioNotSupport
}

func (*StdioEngine) Delete(ctx context.Context, u *URI) error {
	return errStdioNotSupport
}

func (*StdioEngine) DeleteByPrefix(ctx context.Context, u *URI) error {
	return errStdioNotSupport
}

func (*StdioEngine) Size(_ context.Context, u *URI) (int64, error) {
	return 0, errStdioNotSupport
}

func (*StdioEngine) Exists(_ context.Context, u *URI) (bool, error) {
	return true, nil
}

func (*StdioEngine) List(_ context.Context, _ *URI) ([]Info, error) {
	return nil, errStdioNotSupport
}

type NopCloser struct {
	io.Writer
}

func (*NopCloser) Close() error {
	return nil
}

type nopReadAtCloser struct {
	io.Reader
}

func (*nopReadAtCloser) Close() error {
	return nil
}

func (*nopReadAtCloser) ReadAt([]byte, int64) (int, error) {
	return 0, ErrNotSupported
}

func (*nopReadAtCloser) Seek(int64, int) (int64, error) {
	return 0, ErrNotSupported
}
