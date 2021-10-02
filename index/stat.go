package index

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/storage"
)

type Info struct {
	Size int64      `zed:"size"`
	Keys field.List `zed:"keys"`
}

// Stat returns summary information about the microindex at uri.
func Stat(ctx context.Context, engine storage.Engine, uri *storage.URI) (*Info, error) {
	size, err := engine.Size(ctx, uri)
	if err != nil {
		return nil, err
	}
	r, err := NewReaderFromURI(ctx, zed.NewContext(), engine, uri)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return &Info{
		Size: size,
		Keys: r.Keys(),
	}, nil
}
