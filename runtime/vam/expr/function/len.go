package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr/function"
	"github.com/brimdata/zed/vector"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#len
type Len struct {
	zctx *zed.Context
}

func (l *Len) Call(args ...vector.Any) vector.Any {
	val := vector.Under(args[0])
	out := vector.NewIntEmpty(zed.TypeInt64, val.Len(), nil)
	switch typ := val.Type().(type) {
	case *zed.TypeOfNull:
		return vector.NewConst(zed.NewInt64(0), val.Len(), nil)
	case *zed.TypeRecord:
		length := int64(len(typ.Fields))
		return vector.NewConst(zed.NewInt64(length), val.Len(), nil)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeMap:
		for i := uint32(0); i < val.Len(); i++ {
			start, end, _ := vector.ContainerOffset(val, i)
			out.Append(int64(end) - int64(start))
		}
	case *zed.TypeOfString:
		for i := uint32(0); i < val.Len(); i++ {
			s, _ := vector.StringValue(val, i)
			out.Append(int64(len(s)))
		}
	case *zed.TypeOfBytes:
		for i := uint32(0); i < val.Len(); i++ {
			s, _ := vector.BytesValue(val, i)
			out.Append(int64(len(s)))
		}
	case *zed.TypeOfIP:
		for i := uint32(0); i < val.Len(); i++ {
			ip, null := vector.IPValue(val, i)
			if null {
				out.Append(0)
				continue
			}
			out.Append(int64(len(ip.AsSlice())))
		}
	case *zed.TypeOfNet:
		for i := uint32(0); i < val.Len(); i++ {
			n, null := vector.NetValue(val, i)
			if null {
				out.Append(0)
				continue
			}
			out.Append(int64(len(zed.AppendNet(nil, n))))
		}
	case *zed.TypeError:
		return vector.NewWrappedError(l.zctx, "len()", val)
	case *zed.TypeOfType:
		for i := uint32(0); i < val.Len(); i++ {
			v, null := vector.TypeValueValue(val, i)
			if null {
				out.Append(0)
				continue
			}
			t, err := l.zctx.LookupByValue(v)
			if err != nil {
				panic(err)
			}
			out.Append(int64(function.TypeLength(t)))
		}
	default:
		return vector.NewWrappedError(l.zctx, "len: bad type", val)
	}
	return out
}
