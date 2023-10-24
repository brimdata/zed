package vcache

import (
	"fmt"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	meta "github.com/brimdata/zed/vng/vector"
)

func loadVector(any *vector.Any, typ zed.Type, path field.Path, m meta.Metadata, r io.ReaderAt) (vector.Any, error) {
	switch m := m.(type) {
	case *meta.Named:
		return loadVector(any, typ.(*zed.TypeNamed).Type, path, m.Values, r)
	case *meta.Record:
		return loadRecord(any, typ.(*zed.TypeRecord), path, m, r)
	case *meta.Primitive:
		if len(path) != 0 {
			return nil, fmt.Errorf("internal error: vcache encountered path at primitive element: %q", strings.Join(path, "."))
		}
		if *any == nil {
			v, err := loadPrimitive(typ, m, r)
			if err != nil {
				return nil, err
			}
			*any = v
		}
		return *any, nil
	case *meta.Array:
		return loadArray(any, typ, path, m, r)
	case *meta.Set:
		a := *(*meta.Array)(m)
		return loadArray(any, typ, path, &a, r)
	case *meta.Map:
		return loadMap(any, typ, path, m, r)
	case *meta.Union:
		return loadUnion(any, typ.(*zed.TypeUnion), path, m, r)
	case *meta.Nulls:
		return loadNulls(any, typ, path, m, r)
	case *meta.Const:
		return vector.NewConst(m.Value), nil
	default:
		return nil, fmt.Errorf("vector cache: type %T not supported", m)
	}
}
