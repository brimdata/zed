package storage

import (
	"context"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

type Kind int

const (
	KindUnknown Kind = iota
	KindUniZng
	KindArchive
)

func (k Kind) String() string {
	switch k {
	case KindUniZng:
		return "unizng"
	case KindArchive:
		return "archive"
	case KindUnknown:
		fallthrough
	default:
		return "unknown storage kind"
	}
}

type Summary struct {
	Kind      Kind
	Span      nano.Span
	DataBytes int64
}

type Storage interface {
	Open(ctx context.Context, span nano.Span) (zbuf.ReadCloser, error)
	Summary(ctx context.Context) (Summary, error)
	NativeDirection() zbuf.Direction
}
