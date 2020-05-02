package zbuf

import (
	"fmt"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

func NewRawAndTsFromZeekValues(typ *zng.TypeRecord, tsCol int, vals [][]byte) (zcode.Bytes, nano.Ts, error) {
	if nv, nc := len(vals), len(typ.Columns); nv != nc {
		// Don't pass vals to fmt.Errorf or it will escape to the heap.
		return nil, 0, fmt.Errorf("got %d values, expected %d", nv, nc)
	}
	n := 2 // Estimate for descriptor ID uvarint.
	for _, v := range vals {
		n += len(v) + 1 // Estimate for zval and its length uvarint.
	}
	raw := make([]byte, 0, n)
	var ts nano.Ts
	for i, val := range vals {
		var err error
		if i == tsCol {
			ts, err = nano.Parse(val)
			if err != nil {
				return nil, 0, err
			}
		}
		raw = appendZvalFromZeek(raw, typ.Columns[i].Type, val)
	}
	return raw, ts, nil
}
