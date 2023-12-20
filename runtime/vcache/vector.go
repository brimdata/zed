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

type loader struct {
	zctx *zed.Context
	r    io.ReaderAt
}

func (l *loader) loadVector(any *vector.Any, typ zed.Type, path field.Path, m meta.Metadata) (vector.Any, error) {
	switch m := m.(type) {
	case *meta.Named:
		return l.loadVector(any, typ.(*zed.TypeNamed).Type, path, m.Values)
	case *meta.Record:
		return l.loadRecord(any, typ.(*zed.TypeRecord), path, m)
	case *meta.Primitive:
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
	case *meta.Array:
		return l.loadArray(any, typ, path, m)
	case *meta.Set:
		a := *(*meta.Array)(m)
		return l.loadArray(any, typ, path, &a)
	case *meta.Map:
		return l.loadMap(any, typ, path, m)
	case *meta.Union:
		return l.loadUnion(any, typ.(*zed.TypeUnion), path, m)
	case *meta.Nulls:
		return l.loadNulls(any, typ, path, m)
	case *meta.Const:
		*any = vector.NewConst(m.Value, m.Count)
		return *any, nil
	default:
		return nil, fmt.Errorf("vector cache: type %T not supported", m)
	}
}
