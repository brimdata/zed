package zeek

import (
	"fmt"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/pkg/zval"
)

type Flattener struct {
	mapper *resolver.Mapper
}

func NewFlattener() *Flattener {
	return &Flattener{
		mapper: resolver.NewMapper(resolver.NewTable()),
	}
}

func recode(dst zval.Encoding, typ *zeek.TypeRecord, in zval.Encoding) (zval.Encoding, error) {
	if in == nil {
		for k := 0; k < len(typ.Columns); k++ {
			dst = zval.Append(dst, nil, false)
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
		if childType, ok := col.Type.(*zeek.TypeRecord); ok {
			dst, err = recode(dst, childType, val)
			if err != nil {
				return nil, err
			}
		} else {
			dst = zval.Append(dst, val, container)
		}
	}
	return dst, nil
}

func (f *Flattener) Flatten(r *zson.Record) (*zson.Record, error) {
	id := r.Descriptor.ID
	d := f.mapper.Map(id)
	if d == nil {
		cols := flattenColumns(r.Type.Columns)
		outRecord := zeek.LookupTypeRecord(cols)
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
	return zson.NewRecordNoTs(d, zv), nil
}

// flattenColumns turns nested records into a series of columns of
// the form "outer.inner".  XXX It only works for one level of nesting.
func flattenColumns(cols []zeek.Column) []zeek.Column {
	ret := make([]zeek.Column, 0)
	for _, c := range cols {
		recType, isRecord := c.Type.(*zeek.TypeRecord)
		if isRecord {
			for _, inner := range recType.Columns {
				name := fmt.Sprintf("%s.%s", c.Name, inner.Name)
				ret = append(ret, zeek.Column{name, inner.Type})
			}
		} else {
			ret = append(ret, c)
		}
	}
	return ret
}
