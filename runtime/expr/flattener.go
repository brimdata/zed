package expr

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Flattener struct {
	zctx   *zed.Context
	mapper *zed.Mapper
}

// NewFlattener returns a flattener that transforms nested records to flattened
// records where the type context of the received records must match the
// zctx parameter provided here.  Any new type descriptors that are created
// to flatten types also use zctx.
func NewFlattener(zctx *zed.Context) *Flattener {
	return &Flattener{
		zctx: zctx,
		// This mapper maps types back into the same context and gives
		// us a convenient way to track type-ID to type-ID for types that
		// need to be flattened.
		mapper: zed.NewMapper(zctx),
	}
}

func recode(dst zcode.Bytes, typ *zed.TypeRecord, in zcode.Bytes) (zcode.Bytes, error) {
	if in == nil {
		for _, f := range typ.Fields {
			if typ, ok := zed.TypeUnder(f.Type).(*zed.TypeRecord); ok {
				var err error
				dst, err = recode(dst, typ, nil)
				if err != nil {
					return nil, err
				}
			} else {
				dst = zcode.Append(dst, nil)
			}
		}
		return dst, nil
	}
	it := in.Iter()
	fieldno := 0
	for !it.Done() {
		val := it.Next()
		f := typ.Fields[fieldno]
		fieldno++
		if childType, ok := zed.TypeUnder(f.Type).(*zed.TypeRecord); ok {
			var err error
			dst, err = recode(dst, childType, val)
			if err != nil {
				return nil, err
			}
		} else {
			dst = zcode.Append(dst, val)
		}
	}
	return dst, nil
}

func (f *Flattener) Flatten(r zed.Value) (zed.Value, error) {
	id := r.Type().ID()
	flatType := f.mapper.Lookup(id)
	if flatType == nil {
		flatType = f.zctx.MustLookupTypeRecord(FlattenFields(r.Fields()))
		f.mapper.EnterType(id, flatType)
	}
	// Since we are mapping the input context to itself we can do a
	// pointer comparison to see if the types are the same and there
	// is no need to record.
	if zed.TypeUnder(r.Type()) == flatType {
		return r, nil
	}
	zv, err := recode(nil, zed.TypeRecordOf(r.Type()), r.Bytes())
	if err != nil {
		return zed.Null, err
	}
	return zed.NewValue(flatType.(*zed.TypeRecord), zv), nil
}

// FlattenFields turns nested records into a series of fields of
// the form "outer.inner".
func FlattenFields(fields []zed.Field) []zed.Field {
	ret := []zed.Field{}
	for _, f := range fields {
		if recType, ok := zed.TypeUnder(f.Type).(*zed.TypeRecord); ok {
			inners := FlattenFields(recType.Fields)
			for i := range inners {
				inners[i].Name = fmt.Sprintf("%s.%s", f.Name, inners[i].Name)
			}
			ret = append(ret, inners...)
		} else {
			ret = append(ret, f)
		}
	}
	return ret
}
