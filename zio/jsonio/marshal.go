package jsonio

import (
	"encoding/hex"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

func marshalAny(typ zed.Type, bytes zcode.Bytes) interface{} {
	if bytes == nil {
		return nil
	}
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return marshalAny(typ.Type, bytes)
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64:
		return zed.DecodeUint(bytes)
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64:
		return zed.DecodeInt(bytes)
	case *zed.TypeOfDuration:
		return zed.DecodeDuration(bytes).String()
	case *zed.TypeOfTime:
		return zed.DecodeTime(bytes).Time().Format(time.RFC3339Nano)
	case *zed.TypeOfFloat32:
		return zed.DecodeFloat32(bytes)
	case *zed.TypeOfFloat64:
		return zed.DecodeFloat64(bytes)
	case *zed.TypeOfBool:
		return zed.DecodeBool(bytes)
	case *zed.TypeOfBytes:
		return "0x" + hex.EncodeToString(bytes)
	case *zed.TypeOfString:
		return string(bytes)
	case *zed.TypeOfIP:
		return zed.DecodeIP(bytes).String()
	case *zed.TypeOfNet:
		return zed.DecodeNet(bytes).String()
	case *zed.TypeOfNull:
		return nil
	case *zed.TypeRecord:
		return marshalRecord(typ, bytes)
	case *zed.TypeArray:
		return marshalArray(typ, bytes)
	case *zed.TypeSet:
		return marshalSet(typ, bytes)
	case *zed.TypeMap:
		return marshalMap(typ, bytes)
	case *zed.TypeUnion:
		return marshalUnion(typ, bytes)
	case *zed.TypeEnum:
		return marshalEnum(typ, bytes)
	case *zed.TypeError:
		return map[string]interface{}{"error": marshalAny(typ.Type, bytes)}
	default:
		return zson.MustFormatValue(*zed.NewValue(typ, bytes))
	}
}

func marshalRecord(typ *zed.TypeRecord, bytes zcode.Bytes) interface{} {
	it := bytes.Iter()
	rec := make(map[string]interface{})
	for _, col := range typ.Columns {
		rec[col.Name] = marshalAny(col.Type, it.Next())
	}
	return rec
}

func marshalArray(typ *zed.TypeArray, bytes zcode.Bytes) interface{} {
	a := make([]interface{}, 0)
	it := bytes.Iter()
	for !it.Done() {
		a = append(a, marshalAny(typ.Type, it.Next()))
	}
	return a
}

func marshalSet(typ *zed.TypeSet, bytes zcode.Bytes) interface{} {
	s := make([]interface{}, 0)
	it := bytes.Iter()
	for !it.Done() {
		s = append(s, marshalAny(typ.Type, it.Next()))
	}
	return s
}

type Entry struct {
	Key   interface{} `json:"key"`
	Value interface{} `json:"value"`
}

func marshalMap(typ *zed.TypeMap, bytes zcode.Bytes) interface{} {
	var entries []Entry
	it := bytes.Iter()
	for !it.Done() {
		key := marshalAny(typ.KeyType, it.Next())
		val := marshalAny(typ.ValType, it.Next())
		entries = append(entries, Entry{key, val})
	}
	return entries
}

func marshalUnion(typ *zed.TypeUnion, bytes zcode.Bytes) interface{} {
	it := bytes.Iter()
	selector := int(zed.DecodeUint(it.Next()))
	inner, err := typ.Type(selector)
	if err != nil {
		return err
	}
	return marshalAny(inner, it.Next())
}

func marshalEnum(typ *zed.TypeEnum, bytes zcode.Bytes) interface{} {
	selector := int(zed.DecodeUint(bytes))
	if selector >= len(typ.Symbols) {
		return "<bad enum>"
	}
	return typ.Symbols[selector]
}
