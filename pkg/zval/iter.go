package zval

import (
	"fmt"
)

// Iter iterates over a sequence of zval Encodings.
type Iter Encoding

// Done returns true if no zvals remain.
func (i *Iter) Done() bool {
	return len(*i) == 0
}

// Next returns the next zval.  It returns an empty slice for an empty or
// zero-length zval and nil for an unset zval.
func (i *Iter) Next() (Encoding, bool, error) {
	// Uvarint is zero for an unset zval; otherwise, it is the value's
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
	return Encoding(val), tagIsContainer(u64), nil
}
