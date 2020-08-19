package zdx

import (
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zng/resolver"
)

type InfoKey struct {
	Name     string
	TypeName string
}

type Info struct {
	Size int64
	Keys []InfoKey
}

// Stat returns summary information about the microindex at uri.
func Stat(uri iosrc.URI) (*Info, error) {
	si, err := iosrc.Stat(uri)
	if err != nil {
		return nil, err
	}
	size := si.Size()
	r, err := NewReaderFromURI(resolver.NewContext(), uri)
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
