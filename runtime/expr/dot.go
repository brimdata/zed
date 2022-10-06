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

func ValueUnder(val *zed.Value) *zed.Value {
	typ := val.Type
	if _, ok := typ.(*zed.TypeNamed); !ok {
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
		typ, bytes = union.Untag(bytes)
	}
}

func (d *DotExpr) Eval(ectx Context, this *zed.Value) *zed.Value {
	rec := d.record.Eval(ectx, this)
	val := ValueUnder(rec)
	if _, ok := val.Type.(*zed.TypeOfType); ok {
		return d.evalTypeOfType(ectx, val.Bytes)
	}
	if typ, ok := val.Type.(*zed.TypeMap); ok {
		return indexMap(d.zctx, ectx, typ, val.Bytes, zed.NewString(d.field))
	}
	recType, ok := val.Type.(*zed.TypeRecord)
	if !ok {
		return d.zctx.Missing()
	}
	idx, ok := recType.ColumnOfField(d.field)
	if !ok {
		return d.zctx.Missing()
	}
	typ := recType.Columns[idx].Type
	if val.IsNull() {
		// The record is null.  Return null value of the field type.
		return ectx.NewValue(typ, nil)
	}
	//XXX see PR #1071 to improve this (though we need this for Index anyway)
	field := getNthFromContainer(val.Bytes, idx)
	return ectx.NewValue(recType.Columns[idx].Type, field)
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
