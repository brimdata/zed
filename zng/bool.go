package zng

import (
	"github.com/mccanne/zq/zcode"
)

type TypeOfBool struct{}

func NewBool(b bool) Value {
	return Value{TypeBool, EncodeBool(b)}
}

func EncodeBool(b bool) zcode.Bytes {
	var v [1]byte
	if b {
		v[0] = 1
	}
	return v[:]
}

func DecodeBool(zv zcode.Bytes) (bool, error) {
	if zv == nil {
		return false, ErrUnset
	}
	if zv[0] != 0 {
		return true, nil
	}
	return false, nil
}

func (t *TypeOfBool) Parse(in []byte) (zcode.Bytes, error) {
	b, err := UnsafeParseBool(in)
	if err != nil {
		return nil, err
	}
	return EncodeBool(b), nil
}

func (t *TypeOfBool) ID() int {
	return IdBool
}

func (t *TypeOfBool) String() string {
	return "bool"
}

func (t *TypeOfBool) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	b, err := DecodeBool(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	if b {
		return "T"
	}
	return "F"
}

func (t *TypeOfBool) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeBool(zv)
}
