package index

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
)

type InfoKey struct {
	Name     string `zed:"name"`
	TypeName string `zed:"type"`
}

type Info struct {
	Size int64     `zed:"size"`
	Keys []InfoKey `zed:"keys"`
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

	columns := r.Keys().Columns
	keys := make([]InfoKey, len(columns))
	for i, c := range columns {
		keys[i] = InfoKey{
			Name:     c.Name,
			TypeName: c.Type.String(),
		}
	}
	return &Info{
		Size: size,
		Keys: keys,
	}, nil
}
