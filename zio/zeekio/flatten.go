package zeekio

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Flattener struct {
	zctx   *resolver.Context
	mapper *resolver.Mapper
}

// NewFlattener returns a flattener that transforms nested records to flattened
// records where the type context of the received records must match the
// zctx parameter provided here.  Any new type descriptors that are created
// to flatten types also use zctx.
func NewFlattener(zctx *resolver.Context) *Flattener {
	return &Flattener{
		zctx: zctx,
		// This mapper maps types back into the same context and gives
		// us a convenient way to track type-ID to type-ID for types that
		// need to be flattened.
		mapper: resolver.NewMapper(zctx),
	}
}

func recode(dst zcode.Bytes, typ *zng.TypeRecord, in zcode.Bytes) (zcode.Bytes, error) {
	if in == nil {
		for k := 0; k < len(typ.Columns); k++ {
			dst = zcode.AppendPrimitive(dst, nil)
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
		if childType, ok := col.Type.(*zng.TypeRecord); ok {
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

func (f *Flattener) Flatten(r *zng.Record) (*zng.Record, error) {
	id := r.Type.ID()
	flatType := f.mapper.Map(id)
	if flatType == nil {
		cols := flattenColumns(r.Type.Columns)
		flatType = f.zctx.LookupTypeRecord(cols)
		f.mapper.EnterTypeRecord(id, flatType)
	}
	// Since we are mapping the input context to itself we can do a
	// pointer comparison to see if the types are the same and there
	// is no need to record.
	if r.Type == flatType {
		return r, nil
	}
	zv, err := recode(nil, r.Type, r.Raw)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(flatType, zv)

}

// flattenColumns turns nested records into a series of columns of
// the form "outer.inner".  XXX It only works for one level of nesting.
func flattenColumns(cols []zng.Column) []zng.Column {
	ret := make([]zng.Column, 0)
	for _, c := range cols {
		recType, isRecord := c.Type.(*zng.TypeRecord)
		if isRecord {
			for _, inner := range recType.Columns {
				name := fmt.Sprintf("%s.%s", c.Name, inner.Name)
				ret = append(ret, zng.NewColumn(name, inner.Type))
			}
		} else {
			ret = append(ret, c)
		}
	}
	return ret
}
