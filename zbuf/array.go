package zbuf

import (
	"github.com/brimsec/zq/zng"
)

// Array is a slice of of records that implements the Batch interface.
type Array []*zng.Record

func (a Array) Ref() {
	// do nothing... let the GC reclaim it
}

func (a Array) Unref() {
	// do nothing... let the GC reclaim it
}

func (a Array) Length() int {
	return len(a)
}

func (a Array) Records() []*zng.Record {
	return a
}

//XXX should change this to Record()
func (a Array) Index(k int) *zng.Record {
	if k < len(a) {
		return a[k]
	}
	return nil
}

func (a *Array) Append(r *zng.Record) {
	*a = append(*a, r)
}
