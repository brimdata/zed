package zng

import (
	"fmt"

	"github.com/brimsec/zq/zcode"
)

func Split(elemType Type, b zcode.Bytes) ([]Value, error) {
	if b == nil {
		return nil, nil
	}
	vals := []Value{}
	for it := b.Iter(); !it.Done(); {
		zv, _, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("parsing element type '%s' value %q: %w", elemType.ZSON(), zv, err)
		}
		vals = append(vals, Value{elemType, zv})
	}
	return vals, nil
}
