package zcode

import (
	"fmt"
)

// Iter iterates over a sequence of encoded Bytes.
type Iter Bytes

// Done returns true if no zvals remain.
func (i *Iter) Done() bool {
	return len(*i) == 0
}

// Next returns the next value as a Bytes type.  It returns an empty slice for
// an empty or zero-length value and nil for an unset value.
func (i *Iter) Next() (Bytes, bool, error) {
	// Uvarint is zero for an unset value; otherwise, it is the value's
	// length plus one.
	u64, n := Uvarint(*i)
	if n <= 0 {
		return nil, false, fmt.Errorf("bad uvarint: %d", n)
	}
	if tagIsUnset(u64) {
		*i = (*i)[n:]
		return nil, tagIsContainer(u64), nil
	}
	end := n + tagLength(u64)
	val := (*i)[n:end]
	*i = (*i)[end:]
	return Bytes(val), tagIsContainer(u64), nil
}
