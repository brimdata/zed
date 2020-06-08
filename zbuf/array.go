package zbuf

import (
	"github.com/brimsec/zq/zng"
)

// Array is a slice of of records that implements the Batch interface.
type Array struct {
	records []*zng.Record
}

// NewArray returns an Array object holding the passed-in records.
func NewArray(r []*zng.Record) *Array {
	return &Array{
		records: r,
	}
}

func (a *Array) Ref() {
	// do nothing... let the GC reclaim it
}

func (a *Array) Unref() {
	// do nothing... let the GC reclaim it
}

func (a *Array) Length() int {
	return len(a.records)
}

func (a *Array) Records() []*zng.Record {
	return a.records
}

//XXX should change this to Record()
func (a *Array) Index(k int) *zng.Record {
	if k < len(a.records) {
		return a.records[k]
	}
	return nil
}

func (a *Array) Append(r *zng.Record) {
	a.records = append(a.records, r)
}
