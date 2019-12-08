package zson

import (
	"github.com/mccanne/zq/pkg/nano"
)

// Array is a slice of of records that implements the Batch interface.
type Array struct {
	span    nano.Span
	records []*Record
}

// NewArray returns an Array object holding the passed-in records.
func NewArray(r []*Record, s nano.Span) *Array {
	return &Array{
		span:    s,
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

func (a *Array) Records() []*Record {
	return a.records
}

//XXX should update span on drop
func (a *Array) Span() nano.Span {
	return a.span
}

//XXX should change this to Record()
func (a *Array) Index(k int) *Record {
	if k < len(a.records) {
		return a.records[k]
	}
	return nil
}

func (a *Array) Append(r *Record) {
	s := nano.Span{Ts: r.Ts}
	first := a.span == nano.Span{}
	if first {
		a.span = s
	} else {
		a.span = a.span.Union(s)
	}
	a.records = append(a.records, r)
}
