package index

import (
	"context"

	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zng/resolver"
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
func Stat(ctx context.Context, uri iosrc.URI) (*Info, error) {
	si, err := iosrc.Stat(ctx, uri)
	if err != nil {
		return nil, err
	}
	size := si.Size()
	r, err := NewReaderFromURI(ctx, resolver.NewContext(), uri)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	columns := r.Keys().Columns
	keys := make([]InfoKey, len(columns))
	for i, c := range columns {
		keys[i] = InfoKey{
			Name:     c.Name,
			TypeName: c.Type.ZSON(),
		}
	}
	return &Info{
		Size: size,
		Keys: keys,
	}, nil
}
