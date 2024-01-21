package vam

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

//XXX this is just enough of expr to get the hacky agg demos running
// more generalization here is coming soon!

type Evaluator interface {
	Eval(vector.Any) vector.Any
}

type This struct{}

func (*This) Eval(this vector.Any) vector.Any {
	return this
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

func (d *DotExpr) Eval(vec vector.Any) vector.Any {
	switch vec := d.record.Eval(vec).(type) {
	case *vector.Record:
		i, ok := vec.Typ.IndexOfField(d.field)
		if !ok {
			return vector.NewMissing(d.zctx, vec.Len())
		}
		return vec.Fields[i]
	case *vector.Map:
		panic("vam.DotExpr Map TBD")
	case *vector.TypeValue:
		panic("vam.DotExpr TypeValue TBD")
	case *vector.Union:
		vecs := make([]vector.Any, 0, len(vec.Values))
		for _, vec := range vec.Values {
			vecs = append(vecs, d.Eval(vec))
		}
		return vec.Copy(vecs)
	default:
		return vector.NewMissing(d.zctx, vec.Len())
	}
}
