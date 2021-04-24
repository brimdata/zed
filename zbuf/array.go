package zbuf

import (
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
)

// Array is a slice of of records that implements the Batch and
// the Reader interfaces.
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

func (a *Array) Write(r *zng.Record) error {
	a.Append(r)
	return nil
}

// Read returns removes the first element of the Array and returns it,
// or it returns nil if the Array is empty.
func (a *Array) Read() (*zng.Record, error) {
	var rec *zng.Record
	if len(*a) > 0 {
		rec = (*a)[0]
		*a = (*a)[1:]
	}
	return rec, nil
}

func (a Array) NewReader() zio.Reader {
	return &a
}
