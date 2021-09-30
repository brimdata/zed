package order

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/field"
)

var Nil = Layout{}

type Layout struct {
	Order Which      `json:"order" zed:"order"`
	Keys  field.List `json:"keys" zed:"keys"`
}

func (l Layout) Primary() field.Path {
	if len(l.Keys) != 0 {
		return l.Keys[0]
	}
	return nil
}

func (l Layout) IsNil() bool {
	return len(l.Keys) == 0
}

func (l Layout) Equal(to Layout) bool {
	return l.Order == to.Order && l.Keys.Equal(to.Keys)
}

func (l Layout) String() string {
	return fmt.Sprintf("%s:%s", field.List(l.Keys), l.Order)
}

func NewLayout(order Which, keys field.List) Layout {
	return Layout{order, keys}
}

func ParseLayout(s string) (Layout, error) {
	if s == "" {
		return Nil, nil
	}
	which := Asc
	parts := strings.Split(s, ":")
	if len(parts) > 1 {
		if len(parts) > 2 {
			return Nil, errors.New("only one order clause allowed in layout description")
		}
		var err error
		which, err = Parse(parts[1])
		if err != nil {
			return Nil, err
		}
	}
	keys := field.DottedList(parts[0])
	return Layout{Keys: keys, Order: which}, nil
}
