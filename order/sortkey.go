package order

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/pkg/field"
)

type SortKey struct {
	Order Which      `json:"order" zed:"order"`
	Key   field.Path `json:"key" zed:"key"`
}

func (s SortKey) Equal(to SortKey) bool {
	return s.Order == to.Order && s.Key.Equal(to.Key)
}

func (s SortKey) String() string {
	return fmt.Sprintf("%s:%s", s.Key, s.Order)
}

func NewSortKey(order Which, key field.Path) SortKey {
	return SortKey{order, key}
}

type SortKeys []SortKey

func (s SortKeys) Primary() SortKey { return s[0] }
func (s SortKeys) IsNil() bool      { return len(s) == 0 }

func (s SortKeys) Equal(to SortKeys) bool {
	if len(s) != len(to) {
		return false
	}
	for i := range s {
		if !s[i].Equal(to[i]) {
			return false
		}
	}
	return true
}

func ParseSortKeys(s string) (SortKeys, error) {
	if s == "" {
		return nil, nil
	}
	which := Asc
	parts := strings.Split(s, ":")
	if len(parts) > 1 {
		if len(parts) > 2 {
			return nil, errors.New("only one order clause allowed in sortkey description")
		}
		var err error
		which, err = Parse(parts[1])
		if err != nil {
			return nil, err
		}
	}
	keys := field.DottedList(parts[0])
	var sortKeys []SortKey
	for _, k := range keys {
		sortKeys = append(sortKeys, SortKey{Key: k, Order: which})
	}
	return sortKeys, nil
}
