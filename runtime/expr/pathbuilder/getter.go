package pathbuilder

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
)

// A getter provides random access to values in a zcode container
// using zcode.Iter. It uses a cursor to avoid quadratic re-seeks for
// the common case where values are fetched sequentially.
type getter struct {
	cursor int
	bytes  zcode.Bytes
	it     zcode.Iter
}

func newGetter(cont zcode.Bytes) getter {
	return getter{
		cursor: -1,
		bytes:  cont,
		it:     cont.Iter(),
	}
}

func (ig *getter) nth(n int) (zcode.Bytes, error) {
	if n < ig.cursor {
		ig.it = ig.bytes.Iter()
	}
	for !ig.it.Done() {
		zv := ig.it.Next()
		ig.cursor++
		if ig.cursor == n {
			return zv, nil
		}
	}
	return nil, fmt.Errorf("getter.nth: array index %d out of bounds", n)
}
