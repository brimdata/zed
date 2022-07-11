package zbuf

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

// Array is a slice of of records that implements the Batch and
// the Reader interfaces.
type Array struct {
	values []zed.Value
}

var _ Batch = (*Array)(nil)
var _ zio.Reader = (*Array)(nil)
var _ zio.Writer = (*Array)(nil)

//XXX this should take the frame arg too and the procs that create
// new arrays need to propagate their frames downstream.
func NewArray(vals []zed.Value) *Array {
	return &Array{values: vals}
}

func (a *Array) Ref() {
	// do nothing... let the GC reclaim it
}

func (a *Array) Unref() {
	// do nothing... let the GC reclaim it
}

func (a *Array) Values() []zed.Value {
	return a.values
}

func (a *Array) Append(r *zed.Value) {
	a.values = append(a.values, *r)
}

func (a *Array) Vars() []zed.Value {
	// XXX TBD
	return nil
}

func (a *Array) Write(r *zed.Value) error {
	a.Append(r.Copy())
	return nil
}

// Read returns removes the first element of the Array and returns it,
// or it returns nil if the Array is empty.
func (a *Array) Read() (*zed.Value, error) {
	var rec *zed.Value
	if len(a.values) > 0 {
		rec = &a.values[0]
		a.values = a.values[1:]
	}
	return rec, nil
}

func (a Array) NewReader() zio.Reader {
	return &a
}

func (*Array) NewValue(typ zed.Type, bytes zcode.Bytes) *zed.Value {
	// XXX can make this more efficient later
	return zed.NewValue(typ, bytes)
}

func (*Array) CopyValue(val *zed.Value) *zed.Value {
	// XXX can make this more efficient later
	return zed.NewValue(val.Type, val.Bytes)
}
