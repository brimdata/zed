package expr

import (
	"strings"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type FieldExpr struct {
	field  Evaluator
	record Evaluator
	root   bool
}

func NewFieldAccess(s string) *FieldExpr {
	return NewFieldExpr(strings.Split(s, "."))
}

func NewFieldExpr(fields []string) *FieldExpr {
	ret := newFieldNode(fields[0], nil, true)
	for {
		fields = fields[1:]
		if len(fields) == 0 {
			return ret
		}
		ret = newFieldNode(fields[0], ret, false)
	}
}

func newFieldNode(field string, record Evaluator, root bool) *FieldExpr {
	name := &Literal{zng.Value{zng.TypeString, zcode.Bytes(field)}}
	return &FieldExpr{field: name, record: record, root: root}
}

// XXX TODO: change this per https://github.com/brimsec/zq/pull/1071

func accessField(record zng.Value, field string) (zng.Value, error) {
	recType, ok := record.Type.(*zng.TypeRecord)
	if !ok {
		return zng.Value{}, ErrIncompatibleTypes
	}
	idx, ok := recType.ColumnOfField(field)
	if !ok {
		return zng.Value{}, ErrNoSuchField
	}
	typ := recType.Columns[idx].Type
	if record.Bytes == nil {
		// Value was unset.  Return unset value of the indicated type.
		return zng.Value{typ, nil}, nil
	}
	zv, err := getNthFromContainer(record.Bytes, uint(idx))
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{recType.Columns[idx].Type, zv}, nil
}

func (f *FieldExpr) Eval(rec *zng.Record) (zng.Value, error) {
	// XXX make test for 'put x=a[selector]' where selector is another field
	field, err := f.field.Eval(rec)
	if err != nil {
		return zng.Value{}, err
	}
	fid := field.Type.ID()
	if !zng.IsStringy(fid) {
		return zng.Value{}, ErrIncompatibleTypes
	}
	fieldName, _ := zng.DecodeString(field.Bytes)
	if f.record != nil {
		record, err := f.record.Eval(rec)
		if err != nil {
			return zng.Value{}, err
		}
		return accessField(record, fieldName)
	}
	if !f.root {
		return zng.NewString(fieldName), nil
	}
	return accessField(zng.Value{rec.Type, rec.Raw}, fieldName)
}

// FieldExprToString returns ZQL for the Evaluator assuming its a field expr.
func FieldExprToString(e Evaluator) string {
	switch e := e.(type) {
	case *FieldExpr:
		lhs := ""
		if e.record != nil {
			lhs = FieldExprToString(e.record) + "."
		}
		rhs := FieldExprToString(e.field)
		return lhs + rhs
	case *Literal:
		return e.zv.String()
	case *Index:
		lhs := FieldExprToString(e.container)
		rhs := FieldExprToString(e.index)
		return lhs + "[" + rhs + "]"
	}
	return "not a field expr"
}
