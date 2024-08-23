package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#fields
type Fields struct {
	zctx     *zed.Context
	innerTyp *zed.TypeArray
	outerTyp *zed.TypeArray
}

func NewFields(zctx *zed.Context) *Fields {
	inner := zctx.LookupTypeArray(zed.TypeString)
	return &Fields{
		zctx:     zctx,
		innerTyp: inner,
		outerTyp: zctx.LookupTypeArray(inner),
	}
}

func (f *Fields) Call(args ...vector.Any) vector.Any {
	val := vector.Under(args[0])
	switch typ := val.Type().(type) {
	case *zed.TypeRecord:
		paths := buildPath(typ, nil)
		s := vector.NewStringEmpty(val.Len(), nil)
		inOffs, outOffs := []uint32{0}, []uint32{0}
		for i := uint32(0); i < val.Len(); i++ {
			inOffs, outOffs = appendPaths(paths, s, inOffs, outOffs)
		}
		inner := vector.NewArray(f.innerTyp, inOffs, s, nil)
		return vector.NewArray(f.outerTyp, outOffs, inner, nil)
	case *zed.TypeOfType:
		var errs []uint32
		s := vector.NewStringEmpty(val.Len(), nil)
		inOffs, outOffs := []uint32{0}, []uint32{0}
		for i := uint32(0); i < val.Len(); i++ {
			b, _ := vector.TypeValueValue(val, i)
			rtyp := f.recordType(b)
			if rtyp == nil {
				errs = append(errs, i)
				continue
			}
			inOffs, outOffs = appendPaths(buildPath(rtyp, nil), s, inOffs, outOffs)
		}
		inner := vector.NewArray(f.innerTyp, inOffs, s, nil)
		out := vector.NewArray(f.outerTyp, outOffs, inner, nil)
		if len(errs) > 0 {
			return mix(val.Len(), out, errs, vector.NewStringError(f.zctx, "missing", uint32(len(errs))))
		}
		return out
	default:
		return vector.NewStringError(f.zctx, "missing", val.Len())
	}
}

func (f *Fields) recordType(b []byte) *zed.TypeRecord {
	typ, err := f.zctx.LookupByValue(b)
	if err != nil {
		return nil
	}
	rtyp, _ := typ.(*zed.TypeRecord)
	return rtyp
}

func buildPath(typ *zed.TypeRecord, prefix []string) [][]string {
	var out [][]string
	for _, f := range typ.Fields {
		if typ, ok := zed.TypeUnder(f.Type).(*zed.TypeRecord); ok {
			out = append(out, buildPath(typ, append(prefix, f.Name))...)
		} else {
			out = append(out, append(prefix, f.Name))
		}
	}
	return out
}

func appendPaths(paths [][]string, s *vector.String, inner, outer []uint32) ([]uint32, []uint32) {
	for _, path := range paths {
		for _, f := range path {
			s.Append(f)
		}
		inner = append(inner, inner[len(inner)-1]+uint32(len(path)))
	}
	return inner, append(outer, outer[len(outer)-1]+uint32(len(paths)))
}
