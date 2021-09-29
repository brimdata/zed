package parquetio

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

func newData(typ zed.Type, zb zcode.Bytes) (interface{}, error) {
	if zb == nil {
		return nil, nil
	}
	switch typ := zed.AliasOf(typ).(type) {
	case *zed.TypeOfUint8:
		v, err := zed.DecodeUint(zb)
		return uint32(v), err
	case *zed.TypeOfUint16:
		v, err := zed.DecodeUint(zb)
		return uint32(v), err
	case *zed.TypeOfUint32:
		v, err := zed.DecodeUint(zb)
		return uint32(v), err
	case *zed.TypeOfUint64:
		return zed.DecodeUint(zb)
	case *zed.TypeOfInt8:
		v, err := zed.DecodeInt(zb)
		return int32(v), err
	case *zed.TypeOfInt16:
		v, err := zed.DecodeInt(zb)
		return int32(v), err
	case *zed.TypeOfInt32:
		v, err := zed.DecodeInt(zb)
		return int32(v), err
	case *zed.TypeOfInt64, *zed.TypeOfDuration, *zed.TypeOfTime:
		return zed.DecodeInt(zb)
	// XXX add TypeFloat16
	// XXX add TypeFloat32
	case *zed.TypeOfFloat64:
		return zed.DecodeFloat64(zb)
	// XXX add TypeDecimal
	case *zed.TypeOfBool:
		return zed.DecodeBool(zb)
	case *zed.TypeOfBytes, *zed.TypeOfBstring, *zed.TypeOfString:
		return zed.DecodeBytes(zb)
	case *zed.TypeOfIP:
		v, err := zed.DecodeIP(zb)
		return []byte(v.String()), err
	case *zed.TypeOfNet:
		v, err := zed.DecodeNet(zb)
		return []byte(v.String()), err
	case *zed.TypeOfType, *zed.TypeOfError:
		return zed.DecodeBytes(zb)
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
		return []byte(typ.Format(zb)), nil
	case *zed.TypeMap:
		return newMapData(typ.KeyType, typ.ValType, zb)
	}
	panic(fmt.Sprintf("unknown type %T", typ))
}

func newListData(typ zed.Type, zb zcode.Bytes) (map[string]interface{}, error) {
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

func newMapData(keyType, valType zed.Type, zb zcode.Bytes) (map[string]interface{}, error) {
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

func newRecordData(typ *zed.TypeRecord, zb zcode.Bytes) (map[string]interface{}, error) {
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
