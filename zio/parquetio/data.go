package parquetio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

func newData(typ zed.Type, zb zcode.Bytes) (interface{}, error) {
	if zb == nil {
		return nil, nil
	}
	switch typ := zed.TypeUnder(typ).(type) {
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32:
		return uint32(zed.DecodeUint(zb)), nil
	case *zed.TypeOfUint64:
		return zed.DecodeUint(zb), nil
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32:
		return int32(zed.DecodeInt(zb)), nil
	case *zed.TypeOfInt64, *zed.TypeOfDuration, *zed.TypeOfTime:
		return zed.DecodeInt(zb), nil
	// XXX add TypeFloat16
	case *zed.TypeOfFloat32:
		return zed.DecodeFloat32(zb), nil
	case *zed.TypeOfFloat64:
		return zed.DecodeFloat64(zb), nil
	// XXX add TypeDecimal
	case *zed.TypeOfBool:
		return zed.DecodeBool(zb), nil
	case *zed.TypeOfBytes, *zed.TypeOfString:
		return zed.DecodeBytes(zb), nil
	case *zed.TypeOfIP:
		return []byte(zed.DecodeIP(zb).String()), nil
	case *zed.TypeOfNet:
		return []byte(zed.DecodeNet(zb).String()), nil
	case *zed.TypeOfType:
		return zed.DecodeBytes(zb), nil
	case *zed.TypeOfNull:
		return nil, ErrNullType
	case *zed.TypeRecord:
		return newRecordData(typ, zb)
	case *zed.TypeArray:
		return newListData(typ.Type, zb)
	case *zed.TypeSet:
		return newListData(typ.Type, zb)
	case *zed.TypeUnion:
		return nil, ErrUnionType
	case *zed.TypeEnum:
		id := zed.DecodeUint(zb)
		if id >= uint64(len(typ.Symbols)) {
			return nil, errors.New("enum index out of range")
		}
		return []byte(typ.Symbols[id]), nil
	case *zed.TypeMap:
		return newMapData(typ.KeyType, typ.ValType, zb)
	case *zed.TypeError:
		return []byte(zson.String(zed.Value{typ, zb})), nil
	}
	panic(fmt.Sprintf("unknown type %T", typ))
}

func newListData(typ zed.Type, zb zcode.Bytes) (map[string]interface{}, error) {
	var elements []map[string]interface{}
	for it := zb.Iter(); !it.Done(); {
		v, err := newData(typ, it.Next())
		if err != nil {
			return nil, err
		}
		elements = append(elements, map[string]interface{}{"element": v})
	}
	return map[string]interface{}{"list": elements}, nil
}

func newMapData(keyType, valType zed.Type, zb zcode.Bytes) (map[string]interface{}, error) {
	var elements []map[string]interface{}
	for i, it := 0, zb.Iter(); !it.Done(); i++ {
		key, err := newData(keyType, it.Next())
		if err != nil {
			return nil, err
		}
		val, err := newData(valType, it.Next())
		if err != nil {
			return nil, err
		}
		elements = append(elements, map[string]interface{}{
			"key":   key,
			"value": val,
		})
	}
	return map[string]interface{}{"key_value": elements}, nil
}

func newRecordData(typ *zed.TypeRecord, zb zcode.Bytes) (map[string]interface{}, error) {
	m := make(map[string]interface{}, len(typ.Columns))
	for i, it := 0, zb.Iter(); !it.Done(); i++ {
		v, err := newData(typ.Columns[i].Type, it.Next())
		if err != nil {
			return nil, err
		}
		m[typ.Columns[i].Name] = v
	}
	return m, nil
}
