package zbuf

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
)

// Array is a slice of of records that implements the Batch and
// the Reader interfaces.
type Array struct {
	arena  *zed.Arena
	values []zed.Value
}

var _ Batch = (*Array)(nil)
var _ zio.Reader = (*Array)(nil)
var _ zio.Writer = (*Array)(nil)

// XXX this should take the frame arg too and the procs that create
// new arrays need to propagate their frames downstream.
func NewArray(arena *zed.Arena, vals []zed.Value) *Array {
	if arena != nil {
		arena.Ref()
	}
	return &Array{arena, vals}
}

func (a *Array) Ref() {
	// do nothing... let the GC reclaim it
}

func (a *Array) Unref() {
	// do nothing... let the GC reclaim it
}

func (a *Array) Reset() {
	if a.arena != nil {
		a.arena.Reset()
	}
	a.values = a.values[:0]
}

func (a *Array) Values() []zed.Value {
	return a.values
}

func (a *Array) Vars() []zed.Value {
	return nil
}

func (a *Array) Write(r zed.Value) error {
	if a.arena == nil {
		a.arena = zed.NewArena()
	}
	a.values = append(a.values, r.Copy(a.arena))
	return nil
}

func (*Array) Zctx() *zed.Arena { panic("zbuf.Array.Zctx") }

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
