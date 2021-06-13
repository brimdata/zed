package index

import (
	"context"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zson"
)

type InfoKey struct {
	Name     string `zng:"name"`
	TypeName string `zng:"type"`
}

type Info struct {
	Size int64     `zng:"size"`
	Keys []InfoKey `zng:"keys"`
}

// Stat returns summary information about the microindex at uri.
func Stat(ctx context.Context, engine storage.Engine, uri *storage.URI) (*Info, error) {
	size, err := engine.Size(ctx, uri)
	if err != nil {
		return nil, err
	}
	r, err := NewReaderFromURI(ctx, zson.NewContext(), engine, uri)
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
