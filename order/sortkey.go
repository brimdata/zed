package order

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/pkg/field"
)

var Nil = SortKey{}

type SortKey struct {
	Order Which      `json:"order" zed:"order"`
	Keys  field.List `json:"keys" zed:"keys"`
}

func (s SortKey) Primary() field.Path {
	if len(s.Keys) != 0 {
		return s.Keys[0]
	}
	return nil
}

func (s SortKey) IsNil() bool {
	return len(s.Keys) == 0
}

func (s SortKey) Equal(to SortKey) bool {
	return s.Order == to.Order && s.Keys.Equal(to.Keys)
}

func (s SortKey) String() string {
	return fmt.Sprintf("%s:%s", field.List(s.Keys), s.Order)
}

func NewSortKey(order Which, keys field.List) SortKey {
	return SortKey{order, keys}
}

func ParseSortKey(s string) (SortKey, error) {
	if s == "" {
		return Nil, nil
	}
	which := Asc
	parts := strings.Split(s, ":")
	if len(parts) > 1 {
		if len(parts) > 2 {
			return Nil, errors.New("only one order clause allowed in sortkey description")
		}
		var err error
		which, err = Parse(parts[1])
		if err != nil {
			return Nil, err
		}
	}
	keys := field.DottedList(parts[0])
	return SortKey{Keys: keys, Order: which}, nil
}
