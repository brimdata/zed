package parquetio

import (
	"fmt"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type builder struct {
	zcode.Builder
	buf []byte
}

func (b *builder) appendValue(typ zng.Type, v interface{}) {
	switch v := v.(type) {
	case nil:
		b.AppendNull()
	case []byte:
		b.AppendPrimitive(v)
	case bool:
		b.buf = zng.AppendBool(b.buf[:0], v)
		b.AppendPrimitive(b.buf)
	case float32:
		b.buf = zng.AppendFloat64(b.buf[:0], float64(v))
		b.AppendPrimitive(b.buf)
	case float64:
		b.buf = zng.AppendFloat64(b.buf[:0], v)
		b.AppendPrimitive(b.buf)
	case int32:
		b.buf = zng.AppendInt(b.buf[:0], int64(v))
		b.AppendPrimitive(b.buf)
	case int64:
		b.buf = zng.AppendInt(b.buf[:0], v)
		b.AppendPrimitive(b.buf)
	case uint32:
		b.buf = zng.AppendUint(b.buf[:0], uint64(v))
		b.AppendPrimitive(b.buf)
	case uint64:
		b.buf = zng.AppendUint(b.buf[:0], v)
		b.AppendPrimitive(b.buf)
	case map[string]interface{}:
		switch typ := zng.AliasOf(typ).(type) {
		case *zng.TypeArray:
			switch v := v["list"].(type) {
			case nil:
				b.AppendNull()
			case []map[string]interface{}:
				b.BeginContainer()
				for _, m := range v {
					b.appendValue(typ.Type, m["element"])
				}
				b.EndContainer()
			default:
				panic(v)
			}
		case *zng.TypeMap:
			switch v := v["key_value"].(type) {
			case nil:
				b.AppendNull()
			case []map[string]interface{}:
				b.BeginContainer()
				for _, m := range v {
					b.appendValue(typ.KeyType, m["key"])
					b.appendValue(typ.ValType, m["value"])
				}
				b.EndContainer()
			default:
				panic(v)
			}
		case *zng.TypeRecord:
			b.BeginContainer()
			for _, c := range typ.Columns {
				b.appendValue(c.Type, v[c.Name])
			}
			b.EndContainer()
		default:
			panic(typ)
		}
	default:
		panic(fmt.Sprintf("%T", v))
	}
}
