package flattener

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
)

type Flattener struct {
	zctx   *zson.Context
	mapper *resolver.Mapper
}

// New returns a flattener that transforms nested records to flattened
// records where the type context of the received records must match the
// zctx parameter provided here.  Any new type descriptors that are created
// to flatten types also use zctx.
func New(zctx *zson.Context) *Flattener {
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
		for _, col := range typ.Columns {
			if typ, ok := col.Type.(*zng.TypeRecord); ok {
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
	if r.Type == flatType {
		return r, nil
	}
	zv, err := recode(nil, zng.TypeRecordOf(r.Type), r.Bytes)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(flatType.(*zng.TypeRecord), zv), nil
}

// FlattenColumns turns nested records into a series of columns of
// the form "outer.inner".
func FlattenColumns(cols []zng.Column) []zng.Column {
	ret := []zng.Column{}
	for _, c := range cols {
		recType, isRecord := c.Type.(*zng.TypeRecord)
		if isRecord {
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
