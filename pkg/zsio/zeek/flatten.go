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

func NewFlatener() *Flattener {
	return &Flattener{
		mapper: resolver.NewMapper(resolver.NewTable()),
	}
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
	// XXX this loop should build a native zval
	var ss []string
	it := r.ZvalIter()
	for _, col := range r.Descriptor.Type.Columns {
		val, isContainer, err := it.Next()
		if err != nil {
			return nil, err
		}
		recType, isRecord := col.Type.(*zeek.TypeRecord)
		if isRecord {
			it2 := zval.Iter(val)
			for _, inner := range recType.Columns {
				innerVal, isContainer, err := it2.Next()
				if err != nil {
					return nil, err
				}
				ss = append(ss, zson.ZvalToZeekString(inner.Type, innerVal, isContainer))
			}
		} else {
			ss = append(ss, zson.ZvalToZeekString(col.Type, val, isContainer))
		}
	}
	return zson.NewRecordZeekStrings(d, ss...)
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
