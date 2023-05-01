package vcache

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	meta "github.com/brimdata/zed/vng/vector"
)

func loadMap(any *vector.Any, typ zed.Type, path field.Path, m *meta.Map, r io.ReaderAt) (*vector.Map, error) {
	if *any == nil {
		mapType, ok := typ.(*zed.TypeMap)
		if !ok {
			return nil, fmt.Errorf("internal error: vcache.loadMap encountered bad type: %s", typ)
		}
		var keys, values vector.Any
		_, err := loadVector(&keys, mapType.KeyType, path, m.Keys, r)
		if err != nil {
			return nil, err
		}
		_, err = loadVector(&values, mapType.ValType, path, m.Values, r)
		if err != nil {
			return nil, err
		}
		*any = vector.NewMap(mapType, keys, values)
	}
	return (*any).(*vector.Map), nil
}
