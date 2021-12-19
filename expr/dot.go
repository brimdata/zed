package expr

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
)

type This struct{}

func (*This) Eval(this *zed.Value, _ *Scope) *zed.Value {
	return this
}

type DotExpr struct {
	record Evaluator
	field  string
	stash  zed.Value
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

func valOf(val *zed.Value) *zed.Value {
	typ := val.Type
	if _, ok := typ.(*zed.TypeAlias); !ok {
		if _, ok := typ.(*zed.TypeUnion); !ok {
			// common fast path
			return val
		}
	}
	bytes := val.Bytes
	for {
		typ = zed.AliasOf(typ)
		union, ok := typ.(*zed.TypeUnion)
		if !ok {
			return &zed.Value{typ, bytes}
		}
		var err error
		typ, _, bytes, err = union.SplitZNG(bytes)
		if err != nil {
			panic("union split: corrupt Zed bytes")
		}
	}
}

func (d *DotExpr) Eval(this *zed.Value, scope *Scope) *zed.Value {
	rec := d.record.Eval(this, scope)
	val := valOf(rec)
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
		//XXX could lookup singleton
		d.stash = zed.Value{typ, nil}
		return &d.stash
	}
	//XXX see PR #1071 to improve this (though we need this for Index anyway)
	fv, err := getNthFromContainer(val.Bytes, uint(idx))
	if err != nil {
		panic(fmt.Errorf("record field access: corrupt Zed bytes: %w", err))
	}
	d.stash = zed.Value{Type: recType.Columns[idx].Type, Bytes: fv}
	return &d.stash
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
		return field.NewRoot(), nil
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
