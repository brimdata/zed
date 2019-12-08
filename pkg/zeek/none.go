package zeek

import (
	"encoding/json"
	"errors"

	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfNone struct{}

func (t *TypeOfNone) String() string {
	return "none"
}

func (t *TypeOfNone) Parse(value []byte) (string, error) {
	return "none", nil
}

func (t *TypeOfNone) Format(value []byte) (interface{}, error) {
	return "none", nil
}
func (t *TypeOfNone) New(value []byte) (Value, error) {
	return nil, errors.New("cannot call New() on TypeNone")
}

type None struct{}

func (n *None) String() string {
	return "none"
}

func (n None) Encode(dst zval.Encoding) zval.Encoding {
	return zval.AppendValue(dst, []byte("none"))
}

func (n None) Type() Type {
	return TypeNone
}

func (n *None) Comparison(op string) (Predicate, error) {
	return nil, errors.New("cannot compare a none value")
}

func (n *None) Coerce(typ Type) Value {
	return nil
}

func (n *None) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}

func (n None) Elements() ([]Value, bool) { return nil, false }
