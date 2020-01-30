package zng

import (
	"errors"
	"github.com/mccanne/zq/zcode"
)

var ErrInstantiateNull = errors.New("cannot instantiate type null")

type TypeOfNull struct{}

func (t *TypeOfNull) Parse(in []byte) (zcode.Bytes, error) {
	return nil, ErrInstantiateNull
}

func (t *TypeOfNull) ID() int {
	return IdNull
}

func (t *TypeOfNull) String() string {
	return "null"
}

func (t *TypeOfNull) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	return "-"
}

func (t *TypeOfNull) Marshal(zv zcode.Bytes) (interface{}, error) {
	return nil, nil
}
