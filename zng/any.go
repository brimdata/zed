package zng

import (
	"errors"
	"github.com/mccanne/zq/zcode"
)

var ErrInstantiateAny = errors.New("cannot instantiate type any")

type TypeOfAny struct{}

func (t *TypeOfAny) Parse(in []byte) (zcode.Bytes, error) {
	return nil, ErrInstantiateAny
}

func (t *TypeOfAny) ID() int {
	return IdAny
}

func (t *TypeOfAny) String() string {
	return "any"
}

func (t *TypeOfAny) StringOf(zv zcode.Bytes) string {
	return "-"
}

func (t *TypeOfAny) Marshal(zv zcode.Bytes) (interface{}, error) {
	return nil, nil
}
