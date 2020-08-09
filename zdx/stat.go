package zdx

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

type InfoKey struct {
	Name     string
	TypeName string
}

type Info struct {
	Size int64
	Keys []InfoKey
}

// Stat returns summary information about the zdx index at uri.
func Stat(uri iosrc.URI) (*Info, error) {
	var level int
	var size int64
	for {
		si, err := iosrc.Stat(filename(uri, level))
		if err != nil {
			if errors.Is(err, zqe.E(zqe.NotFound)) {
				break
			}
			return nil, err
		}
		level++
		size += si.Size()
	}
	if level == 0 {
		return nil, zqe.E(zqe.NotFound)
	}
	r, err := newReader(resolver.NewContext(), uri, 0)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	rec, err := r.Read()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		// files exists but is empty
		return nil, fmt.Errorf("%s: cannnot read zdx header", uri)
	}
	_, keysType, err := ParseHeader(rec)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", uri, err)
	}
	keys := make([]InfoKey, len(keysType.Columns))
	for i, c := range keysType.Columns {
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
