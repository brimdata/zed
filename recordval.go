package zed

import (
	"errors"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

var (
	ErrMissingField  = errors.New("record missing a field")
	ErrExtraField    = errors.New("record with extra field")
	ErrNotContainer  = errors.New("expected container type, got primitive")
	ErrNotPrimitive  = errors.New("expected primitive type, got container")
	ErrTypeIDInvalid = errors.New("zng type ID out of range")
	ErrBadValue      = errors.New("malformed zng value")
	ErrBadFormat     = errors.New("malformed zng record")
	ErrTypeMismatch  = errors.New("type/value mismatch")
)

func (r *Value) HasField(field string) bool {
	return TypeRecordOf(r.Type).HasField(field)
}

// Walk traverses a value in depth-first order, calling a
// Visitor on the way.
func (r *Value) Walk(rv Visitor) error {
	return Walk(r.Type, r.Bytes, rv)
}

func (r *Value) nth(column int) zcode.Bytes {
	var zv zcode.Bytes
	for i, it := 0, r.Bytes.Iter(); i <= column; i++ {
		if it.Done() {
			return nil
		}
		zv = it.Next()
	}
	return zv
}

func (r *Value) Columns() []Column {
	return TypeRecordOf(r.Type).Columns
}

func (v *Value) DerefByColumn(col int) *Value {
	if v != nil {
		if bytes := v.nth(col); bytes != nil {
			v = &Value{v.Columns()[col].Type, bytes}
		} else {
			v = nil
		}
	}
	return v
}

func (r *Value) ColumnOfField(field string) (int, bool) {
	return TypeRecordOf(r.Type).ColumnOfField(field)
}

func (r *Value) TypeOfColumn(col int) Type {
	return TypeRecordOf(r.Type).Columns[col].Type
}

func (v *Value) Deref(field string) *Value {
	if v == nil {
		return nil
	}
	col, ok := v.ColumnOfField(field)
	if !ok {
		return nil
	}
	return v.DerefByColumn(col)
}

func (v *Value) DerefPath(path field.Path) *Value {
	for len(path) != 0 {
		v = v.Deref(path[0])
		path = path[1:]
	}
	return v
}

func (v *Value) AsString() string {
	if v != nil && TypeUnder(v.Type) == TypeString {
		return DecodeString(v.Bytes)
	}
	return ""
}

func (v *Value) AsBool() bool {
	if v != nil && TypeUnder(v.Type) == TypeBool {
		return DecodeBool(v.Bytes)
	}
	return false
}

func (v *Value) AsInt() int64 {
	if v != nil {
		switch TypeUnder(v.Type).(type) {
		case *TypeOfUint8, *TypeOfUint16, *TypeOfUint32:
			return int64(DecodeUint(v.Bytes))
		case *TypeOfUint64:
			return int64(DecodeUint(v.Bytes))
		case *TypeOfInt8, *TypeOfInt16, *TypeOfInt32, *TypeOfInt64:
			return DecodeInt(v.Bytes)
		}
	}
	return 0
}

func (v *Value) AsTime() nano.Ts {
	if v != nil && TypeUnder(v.Type) == TypeTime {
		return DecodeTime(v.Bytes)
	}
	return 0
}

func (v *Value) MissingAsNull() *Value {
	if v == nil {
		v = Null
	}
	return v
}
