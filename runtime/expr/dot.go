package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/zcode"
)

type This struct{}

func (*This) Eval(_ Context, this *zed.Value) *zed.Value {
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

func NewDottedExpr(zctx *zed.Context, f field.Path) Evaluator {
	ret := Evaluator(&This{})
	for _, name := range f {
		ret = NewDotExpr(zctx, ret, name)
	}
	return ret
}

func (d *DotExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	val := d.record.Eval(ectx, this).Under()
	// Cases are ordered by decreasing expected frequency.
	switch typ := val.Type.(type) {
	case *zed.TypeRecord:
		i, ok := typ.ColumnOfField(d.field)
		if !ok {
			return d.zctx.Missing()
		}
		return ectx.NewValue(typ.Fields[i].Type, getNthFromContainer(val.Bytes(), i))
	case *zed.TypeMap:
		return indexMap(d.zctx, ectx, typ, val.Bytes(), zed.NewString(d.field))
	case *zed.TypeOfType:
		return d.evalTypeOfType(ectx, val.Bytes())
	}
	return d.zctx.Missing()
}

func (d *DotExpr) evalTypeOfType(ectx Context, b zcode.Bytes) *zed.Value {
	typ, _ := d.zctx.DecodeTypeValue(b)
	if typ, ok := zed.TypeUnder(typ).(*zed.TypeRecord); ok {
		if typ, ok := typ.TypeOfField(d.field); ok {
			return d.zctx.LookupTypeValue(typ)
		}
	}
	return d.zctx.Missing()
}

// DotExprToString returns Zed for the Evaluator assuming it's a field expr.
func DotExprToString(e Evaluator) (string, error) {
	f, err := DotExprToField(e)
	if err != nil {
		return "", err
	}
	return f.String(), nil
}

func DotExprToField(e Evaluator) (field.Path, error) {
	switch e := e.(type) {
	case *This:
		return field.Path{}, nil
	case *DotExpr:
		lhs, err := DotExprToField(e.record)
		if err != nil {
			return nil, err
		}
		return append(lhs, e.field), nil
	case *Literal:
		return field.Path{e.val.String()}, nil
	case *Index:
		lhs, err := DotExprToField(e.container)
		if err != nil {
			return nil, err
		}
		rhs, err := DotExprToField(e.index)
		if err != nil {
			return nil, err
		}
		return append(lhs, rhs...), nil
	}
	return nil, errors.New("not a field")
}
