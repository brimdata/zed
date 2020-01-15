package zeekio

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Flattener struct {
	mapper *resolver.Mapper
}

func NewFlattener() *Flattener {
	return &Flattener{
		mapper: resolver.NewMapper(resolver.NewContext()),
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
	id := r.Type.ID
	outputType := f.mapper.Map(id)
	if outputType == nil {
		cols := flattenColumns(r.Type.Columns)
		outputType = f.mapper.EnterByColumns(id, cols)
	}
	// Since we are mapping the input context to itself we can do a
	// pointer comparison to see if the types are the same and there
	// is no need to record.
	if r.Type == outputType {
		return r, nil
	}
	zv, err := recode(nil, r.Type, r.Raw)
	if err != nil {
		return nil, err
	}
	out := zng.NewRecordNoTs(outputType, zv)
	return out, nil

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
