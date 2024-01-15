package vcache

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
)

func (l *loader) loadMap(any *vector.Any, typ zed.Type, path field.Path, m *vng.Map) (*vector.Map, error) {
	if *any == nil {
		mapType, ok := typ.(*zed.TypeMap)
		if !ok {
			return nil, fmt.Errorf("internal error: vcache.loadMap encountered bad type: %s", typ)
		}
		lengths, err := vng.ReadIntVector(m.Lengths, l.r)
		if err != nil {
			return nil, err
		}
		var keys, values vector.Any
		_, err = l.loadVector(&keys, mapType.KeyType, path, m.Keys)
		if err != nil {
			return nil, err
		}
		_, err = l.loadVector(&values, mapType.ValType, path, m.Values)
		if err != nil {
			return nil, err
		}
		*any = vector.NewMap(mapType, lengths, keys, values)
	}
	return (*any).(*vector.Map), nil
}
