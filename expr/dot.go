package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
)

type RootRecord struct{}

func (r *RootRecord) Eval(rec *zed.Record) (zed.Value, error) {
	return rec.Value, nil
}

type DotExpr struct {
	record Evaluator
	field  string
}

func NewDotAccess(record Evaluator, field string) *DotExpr {
	return &DotExpr{record, field}
}

func NewDotExpr(f field.Path) Evaluator {
	ret := Evaluator(&RootRecord{})
	for _, name := range f {
		ret = &DotExpr{ret, name}
	}
	return ret
}

func valOf(zv zed.Value) (zed.Value, error) {
	typ := zv.Type
	bytes := zv.Bytes
	for {
		typ = zed.AliasOf(typ)
		union, ok := typ.(*zed.TypeUnion)
		if !ok {
			return zed.Value{typ, bytes}, nil
		}
		var err error
		typ, _, bytes, err = union.SplitZng(bytes)
		if err != nil {
			return zed.Value{}, err
		}
	}
}

func accessField(zv zed.Value, field string) (zed.Value, error) {
	zv, err := valOf(zv)
	if err != nil {
		return zed.Value{}, err
	}
	recType, ok := zv.Type.(*zed.TypeRecord)
	if !ok {
		return zed.Value{}, zed.ErrMissing
	}
	idx, ok := recType.ColumnOfField(field)
	if !ok {
		return zed.Value{}, zed.ErrMissing
	}
	typ := recType.Columns[idx].Type
	if zv.Bytes == nil {
		// Value was unset.  Return unset value of the indicated type.
		return zed.Value{typ, nil}, nil
	}
	//XXX see PR #1071 to improve this (though we need this for Index anyway)
	fv, err := getNthFromContainer(zv.Bytes, uint(idx))
	if err != nil {
		return zed.Value{}, err
	}
	return zed.Value{recType.Columns[idx].Type, fv}, nil
}

func (f *DotExpr) Eval(rec *zed.Record) (zed.Value, error) {
	lval, err := f.record.Eval(rec)
	if err != nil {
		return zed.Value{}, err
	}
	return accessField(lval, f.field)
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
	case *RootRecord:
		return field.NewRoot(), nil
	case *DotExpr:
		lhs, err := DotExprToField(e.record)
		if err != nil {
			return nil, err
		}
		return append(lhs, e.field), nil
	case *Literal:
		return field.New(e.zv.String()), nil
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
