package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#len
type Map struct {
	zctx *zed.Context
}

var _ Interface = (*Map)(nil)

func (m *Map) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
        switch typ := zed.TypeUnder(args[0].Type).(type) {
	case *zed.TypeOfNull:
                t := m.zctx.LookupTypeMap(zed.TypeNull, zed.TypeNull)
                return zed.NewValue(t, nil)
	case *zed.TypeRecord:
                for
		length = len(typ.Columns)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeMap:
		var err error
		length, err = val.ContainerLength()
		if err != nil {
			panic(err)
		}
	case *zed.TypeOfBytes, *zed.TypeOfString, *zed.TypeOfIP, *zed.TypeOfNet:
		length = len(val.Bytes)
	case *zed.TypeError:
		return l.zctx.WrapError("len()", &val)
	case *zed.TypeOfType:
		t, err := l.zctx.LookupByValue(val.Bytes)
		if err != nil {
			return newError(l.zctx, ectx, err)
		}
		length = typeLength(t)
	default:
		return l.zctx.NewErrorf("len: bad type: %s", zson.FormatType(typ))
	}
	return newInt64(ectx, int64(length))
}

//XXX move this to expr/map?

type zedMap struct {
        keys map[zed.Type]mapEntry
        vals map[zed.Type]struct{}
}

type mapEntry map[string]*zed.Value

func newMap() *zedMap {
        return &zedMap{
                keys: make(map[zed.Type]mapEntry),
                valTypes: make(map[zed.Type]struct{}),
        }
}

func (m *zedMap) Enter(key, val *zed.Value) {
        m.vals[val.Type] = struct{}{}
        entryMap := m.keys[key.Type]
        if entryMap == nil {
                entryMap = make(mapEntry)
                m.keys[key.Type] = entryMap
        }
        entryMap[string(key.Bytes)] = val
}
