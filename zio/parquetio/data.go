package parquetio

import (
	"fmt"

	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zng"
)

func newData(typ zng.Type, zb zcode.Bytes) (interface{}, error) {
	if zb == nil {
		return nil, nil
	}
	switch typ := zng.AliasOf(typ).(type) {
	case *zng.TypeOfUint8:
		v, err := zng.DecodeUint(zb)
		return uint32(v), err
	case *zng.TypeOfUint16:
		v, err := zng.DecodeUint(zb)
		return uint32(v), err
	case *zng.TypeOfUint32:
		v, err := zng.DecodeUint(zb)
		return uint32(v), err
	case *zng.TypeOfUint64:
		return zng.DecodeUint(zb)
	case *zng.TypeOfInt8:
		v, err := zng.DecodeInt(zb)
		return int32(v), err
	case *zng.TypeOfInt16:
		v, err := zng.DecodeInt(zb)
		return int32(v), err
	case *zng.TypeOfInt32:
		v, err := zng.DecodeInt(zb)
		return int32(v), err
	case *zng.TypeOfInt64, *zng.TypeOfDuration, *zng.TypeOfTime:
		return zng.DecodeInt(zb)
	// XXX add TypeFloat16
	// XXX add TypeFloat32
	case *zng.TypeOfFloat64:
		return zng.DecodeFloat64(zb)
	// XXX add TypeDecimal
	case *zng.TypeOfBool:
		return zng.DecodeBool(zb)
	case *zng.TypeOfBytes, *zng.TypeOfBstring, *zng.TypeOfString:
		return zng.DecodeBytes(zb)
	case *zng.TypeOfIP:
		v, err := zng.DecodeIP(zb)
		return []byte(v.String()), err
	case *zng.TypeOfNet:
		v, err := zng.DecodeNet(zb)
		return []byte(v.String()), err
	case *zng.TypeOfType, *zng.TypeOfError:
		return zng.DecodeBytes(zb)
	case *zng.TypeOfNull:
		return nil, ErrNullType
	case *zng.TypeRecord:
		return newRecordData(typ, zb)
	case *zng.TypeArray:
		return newListData(typ.Type, zb)
	case *zng.TypeSet:
		return newListData(typ.Type, zb)
	case *zng.TypeUnion:
		return nil, ErrUnionType
	case *zng.TypeEnum:
		return []byte(typ.ZSONOf(zb)), nil
	case *zng.TypeMap:
		return newMapData(typ.KeyType, typ.ValType, zb)
	}
	panic(fmt.Sprintf("unknown type %T", typ))
}

func newListData(typ zng.Type, zb zcode.Bytes) (map[string]interface{}, error) {
	var elements []map[string]interface{}
	for it := zb.Iter(); !it.Done(); {
		zb2, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		v, err := newData(typ, zb2)
		if err != nil {
			return nil, err
		}
		elements = append(elements, map[string]interface{}{"element": v})
	}
	return map[string]interface{}{"list": elements}, nil
}

func newMapData(keyType, valType zng.Type, zb zcode.Bytes) (map[string]interface{}, error) {
	var elements []map[string]interface{}
	for i, it := 0, zb.Iter(); !it.Done(); i++ {
		keyBytes, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		key, err := newData(keyType, keyBytes)
		if err != nil {
			return nil, err
		}
		valBytes, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		val, err := newData(valType, valBytes)
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

func newRecordData(typ *zng.TypeRecord, zb zcode.Bytes) (map[string]interface{}, error) {
	m := make(map[string]interface{}, len(typ.Columns))
	for i, it := 0, zb.Iter(); !it.Done(); i++ {
		zb2, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		v, err := newData(typ.Columns[i].Type, zb2)
		if err != nil {
			return nil, err
		}
		m[typ.Columns[i].Name] = v
	}
	return m, nil
}
