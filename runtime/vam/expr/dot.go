package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

type This struct{}

func (*This) Eval(val vector.Any) (vector.Any, vector.Any) {
	return val, nil
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

func (d *DotExpr) Eval(val vector.Any) (vector.Any, vector.Any) {
	val, err := d.record.Eval(val)
	switch val := val.(type) {
	case nil:
		return nil, err
	case *vector.Record:
		i, ok := val.Typ.IndexOfField(d.field)
		if !ok {
			return nil, vector.NewMissing(d.zctx, val.Len())
		}
		return val.Fields[i], nil
	case *vector.TypeValue:
		panic("vam.DotExpr TypeValue TBD")
	case *vector.Map:
		panic("vam.DotExpr Map TBD")
	case *vector.Union:
		vals := make([]vector.Any, 0, len(val.Values))
		var errs []vector.Any
		for _, val := range val.Values {
			//XXX blend errors
			val, err := d.Eval(val)
			vals = append(vals, val)
			if err != nil {
				errs = append(errs, err)
			}
		}
		return val.Copy(vals), blendErrors(errs)
	default:
		return nil, vector.NewMissing(d.zctx, val.Len())
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
