package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
)

type This struct{}

func (*This) Eval(val vector.Any) vector.Any {
	return val
}

type DotExpr struct {
	zctx   *zed.Context
	record Evaluator
	field  string
}

func NewDotExpr(zctx *zed.Context, record Evaluator, field string) *DotExpr {
	return &DotExpr{
		zctx:   zctx,
		record: record,
		field:  field,
	}
}

func NewDottedExpr(zctx *zed.Context, f field.Path) Evaluator {
	ret := Evaluator(&This{})
	for _, name := range f {
		ret = NewDotExpr(zctx, ret, name)
	}
	return ret
}

func (d *DotExpr) Eval(val vector.Any) vector.Any {
	switch val := d.record.Eval(val).(type) {
	case nil: //XXX
		return nil
	case *vector.Record:
		i, ok := val.Typ.IndexOfField(d.field)
		if !ok {
			return vector.NewMissing(d.zctx, val.Len())
		}
		return val.Fields[i]
	case *vector.TypeValue:
		panic("vam.DotExpr TypeValue TBD")
	case *vector.Map:
		panic("vam.DotExpr Map TBD")
	case *vector.Union:
		vals := make([]vector.Any, 0, len(val.Values))
		for _, val := range val.Values {
			//XXX blend errors... we need a generic rollup that
			// would take any child variants and flatten them so that
			// a variant always appears at top level, e.g., in a union
			// with embedded variant(s) we need to pop the variant above
			// the union and create different versions of top-level types
			// reflecting the new mix of consituent union components.
			vals = append(vals, d.Eval(val))
		}
		return val.Copy(vals)
	default:
		return vector.NewMissing(d.zctx, val.Len())
	}
}

// XXX
func blendErrors(errs []vector.Any) vector.Any {
	if len(errs) == 0 {
		return nil
	}
	//XXX TBD
	return nil
}
