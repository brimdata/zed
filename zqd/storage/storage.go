package storage

import (
	"context"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

type Kind int

const (
	UnknownStore Kind = iota
	FileStore
	ArchiveStore
)

func (k Kind) String() string {
	switch k {
	case FileStore:
		return "filestore"
	case ArchiveStore:
		return "archivestore"
	case UnknownStore:
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
