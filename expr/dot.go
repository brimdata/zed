package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
)

type This struct{}

func (*This) Eval(_ Context, this *zed.Value) *zed.Value {
	return this
}

type DotExpr struct {
	record Evaluator
	field  string
}

func NewDotExpr(record Evaluator, field string) *DotExpr {
	return &DotExpr{
		record: record,
		field:  field,
	}
}

func NewDottedExpr(f field.Path) Evaluator {
	ret := Evaluator(&This{})
	for _, name := range f {
		ret = NewDotExpr(ret, name)
	}
	return ret
}

func ValueOf(val *zed.Value) *zed.Value {
	typ := val.Type
	if _, ok := typ.(*zed.TypeAlias); !ok {
		if _, ok := typ.(*zed.TypeUnion); !ok {
			// common fast path
			return val
		}
	}
	bytes := val.Bytes
	for {
		typ = zed.TypeUnder(typ)
		union, ok := typ.(*zed.TypeUnion)
		if !ok {
			return &zed.Value{typ, bytes}
		}
		var err error
		typ, _, bytes, err = union.SplitZNG(bytes)
		if err != nil {
			panic(err)
		}
	}
}

func (d *DotExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	rec := d.record.Eval(ectx, this)
	val := ValueOf(rec)
	recType, ok := val.Type.(*zed.TypeRecord)
	if !ok {
		return zed.Missing
	}
	idx, ok := recType.ColumnOfField(d.field)
	if !ok {
		return zed.Missing
	}
	typ := recType.Columns[idx].Type
	if val.IsNull() {
		// The record is null.  Return null value of the field type.
		return ectx.NewValue(typ, nil)
	}
	//XXX see PR #1071 to improve this (though we need this for Index anyway)
	field := getNthFromContainer(val.Bytes, uint(idx))
	return ectx.NewValue(recType.Columns[idx].Type, field)
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
		return field.NewEmpty(), nil
	case *DotExpr:
		lhs, err := DotExprToField(e.record)
		if err != nil {
			return nil, err
		}
		return append(lhs, e.field), nil
	case *Literal:
		return field.New(e.val.String()), nil
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
