package zed

import (
	"github.com/brimdata/zed/zcode"
)

func Split(elemType Type, b zcode.Bytes) ([]Value, error) {
	// We test for nil explicitly and initialize vals to a non-empty
	// slice of size 0 so that we can differentiate between a non-nil
	// but empty Zed container vs a nil Zed container value.
	if b == nil {
		return nil, nil
	}
	vals := []Value{}
	for it := b.Iter(); !it.Done(); {
		zv, _ := it.Next()
		vals = append(vals, Value{elemType, zv})
	}
	return vals, nil
}
