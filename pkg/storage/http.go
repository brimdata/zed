package storage

import (
	"context"
	"errors"
	"io"
)

type HTTPEngine struct{}

var _ Engine = (*HTTPEngine)(nil)

func NewHTTP() *HTTPEngine {
	return &HTTPEngine{}
}

func (*HTTPEngine) Get(_ context.Context, u *URI) (Reader, error) {
	return nil, errors.New("see issue #734")
}

func (*HTTPEngine) Put(_ context.Context, u *URI) (io.WriteCloser, error) {
	return nil, ErrNotSupported
}

func (*HTTPEngine) PutIfNotExists(context.Context, *URI, []byte) error {
	return ErrNotSupported
}

func (*HTTPEngine) Delete(_ context.Context, u *URI) error {
	return ErrNotSupported
}

func (*HTTPEngine) DeleteByPrefix(_ context.Context, u *URI) error {
	return ErrNotSupported
}

func (*HTTPEngine) Size(_ context.Context, u *URI) (int64, error) {
	return 0, ErrNotSupported
}

func (*HTTPEngine) Exists(_ context.Context, u *URI) (bool, error) {
	return false, ErrNotSupported
}

func (*HTTPEngine) List(ctx context.Context, u *URI) ([]Info, error) {
	return nil, ErrNotSupported
}
