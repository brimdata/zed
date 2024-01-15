package vcache

import (
	"fmt"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
)

type loader struct {
	zctx *zed.Context
	r    io.ReaderAt
}

func (l *loader) loadVector(any *vector.Any, typ zed.Type, path field.Path, m vng.Metadata) (vector.Any, error) {
	switch m := m.(type) {
	case *vng.Named:
		return l.loadVector(any, typ.(*zed.TypeNamed).Type, path, m.Values)
	case *vng.Record:
		return l.loadRecord(any, typ.(*zed.TypeRecord), path, m)
	case *vng.Primitive:
		if len(path) != 0 {
			return nil, fmt.Errorf("internal error: vcache encountered path at primitive element: %q", strings.Join(path, "."))
		}
		if *any == nil {
			v, err := l.loadPrimitive(typ, m)
			if err != nil {
				return nil, err
			}
			*any = v
		}
		return *any, nil
	case *vng.Array:
		return l.loadArray(any, typ, path, m)
	case *vng.Set:
		a := *(*vng.Array)(m)
		return l.loadArray(any, typ, path, &a)
	case *vng.Map:
		return l.loadMap(any, typ, path, m)
	case *vng.Union:
		return l.loadUnion(any, typ.(*zed.TypeUnion), path, m)
	case *vng.Nulls:
		return l.loadNulls(any, typ, path, m)
	case *vng.Const:
		*any = vector.NewConst(m.Value, m.Count)
		return *any, nil
	default:
		return nil, fmt.Errorf("vector cache: type %T not supported", m)
	}
}
