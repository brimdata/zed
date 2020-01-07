package zeekio

import (
	"fmt"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Flattener struct {
	mapper *resolver.Mapper
}

func NewFlattener() *Flattener {
	return &Flattener{
		mapper: resolver.NewMapper(resolver.NewTable()),
	}
}

func recode(dst zcode.Bytes, typ *zng.TypeRecord, in zcode.Bytes) (zcode.Bytes, error) {
	if in == nil {
		for k := 0; k < len(typ.Columns); k++ {
			dst = zcode.Append(dst, nil, false)
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
			dst = zcode.Append(dst, val, container)
		}
	}
	return dst, nil
}

func (f *Flattener) Flatten(r *zbuf.Record) (*zbuf.Record, error) {
	id := r.Descriptor.ID
	d := f.mapper.Map(id)
	if d == nil {
		cols := flattenColumns(r.Type.Columns)
		outRecord := zng.LookupTypeRecord(cols)
		d = f.mapper.Enter(id, outRecord)
	}
	if d.Type == r.Descriptor.Type {
		r.Descriptor = d
		return r, nil
	}
	zv, err := recode(nil, r.Descriptor.Type, r.Raw)
	if err != nil {
		return nil, err
	}
	return zbuf.NewRecordNoTs(d, zv), nil
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
				ret = append(ret, zng.Column{name, inner.Type})
			}
		} else {
			ret = append(ret, c)
		}
	}
	return ret
}
