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
		for _, col := range typ.Columns {
			if typ, ok := zed.AliasOf(col.Type).(*zed.TypeRecord); ok {
				var err error
				dst, err = recode(dst, typ, nil)
				if err != nil {
					return nil, err
				}
			} else {
				dst = zcode.AppendNull(dst)
			}
		}
		return dst, nil
	}
	it := in.Iter()
	colno := 0
	for !it.Done() {
		val, container, err := it.Next()
		if err != nil {
			return nil, err
		}
		col := typ.Columns[colno]
		colno++
		if childType, ok := zed.AliasOf(col.Type).(*zed.TypeRecord); ok {
			dst, err = recode(dst, childType, val)
			if err != nil {
				return nil, err
			}
		} else {
			if container {
				dst = zcode.AppendContainer(dst, val)
			} else {
				dst = zcode.AppendPrimitive(dst, val)
			}
		}
	}
	return dst, nil
}

func (f *Flattener) Flatten(r *zed.Value) (*zed.Value, error) {
	id := r.Type.ID()
	flatType := f.mapper.Lookup(id)
	if flatType == nil {
		cols := FlattenColumns(r.Columns())
		var err error
		flatType, err = f.zctx.LookupTypeRecord(cols)
		if err != nil {
			return nil, err
		}
		f.mapper.EnterType(id, flatType)
	}
	// Since we are mapping the input context to itself we can do a
	// pointer comparison to see if the types are the same and there
	// is no need to record.
	if zed.AliasOf(r.Type) == flatType {
		return r, nil
	}
	zv, err := recode(nil, zed.TypeRecordOf(r.Type), r.Bytes)
	if err != nil {
		return nil, err
	}
	return zed.NewValue(flatType.(*zed.TypeRecord), zv), nil
}

// FlattenColumns turns nested records into a series of columns of
// the form "outer.inner".
func FlattenColumns(cols []zed.Column) []zed.Column {
	ret := []zed.Column{}
	for _, c := range cols {
		if recType, ok := zed.AliasOf(c.Type).(*zed.TypeRecord); ok {
			inners := FlattenColumns(recType.Columns)
			for i := range inners {
				inners[i].Name = fmt.Sprintf("%s.%s", c.Name, inners[i].Name)
			}
			ret = append(ret, inners...)
		} else {
			ret = append(ret, c)
		}
	}
	return ret
}
