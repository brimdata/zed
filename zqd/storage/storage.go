package storage

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
)

type Kind string

const (
	UnknownStore Kind = ""
	FileStore    Kind = "filestore"
	ArchiveStore Kind = "archivestore"
)

func (k Kind) String() string {
	return string(k)
}

func (k *Kind) Set(x string) error {
	kx := Kind(x)
	switch kx {
	case FileStore:
		fallthrough
	case ArchiveStore:
		*k = kx
		return nil

	default:
		return fmt.Errorf("unknown storage kind: %s", x)
	}
}

type Config struct {
	Kind    Kind           `json:"kind"`
	Archive *ArchiveConfig `json:"archivestore"`
}

type ArchiveConfig struct {
	OpenOptions   *ArchiveOpenOptions   `json:"open_options"`
	CreateOptions *ArchiveCreateOptions `json:"create_options"`
}

type ArchiveOpenOptions struct {
	LogFilter []string `json:"log_filter"`
}

type ArchiveCreateOptions struct {
	LogSizeThreshold *int64 `json:"log_size_threshold"`
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
