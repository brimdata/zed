package vcache

import (
	"io"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zst"
	"github.com/brimdata/zed/zst/vector"
)

type Array struct {
	segmap  []vector.Segment
	values  Vector
	lengths []int32
}

func NewArray(array *vector.Array, r io.ReaderAt) (*Array, error) {
	values, err := NewVector(array.Values, r)
	if err != nil {
		return nil, err
	}
	return &Array{
		segmap: array.Lengths,
		values: values,
	}, nil
}

func (a *Array) NewIter(reader io.ReaderAt) (iterator, error) {
	// The lengths vector is typically large and is loaded on demand.
	if a.lengths == nil {
		lengths, err := zst.ReadIntVector(a.segmap, reader)
		if err != nil {
			return nil, err
		}
		a.lengths = lengths
	}
	values, err := a.values.NewIter(reader)
	if err != nil {
		return nil, err
	}
	off := 0
	return func(b *zcode.Builder) error {
		b.BeginContainer()
		len := a.lengths[off]
		off++
		for ; len > 0; len-- {
			if err := values(b); err != nil {
				return err
			}
		}
		b.EndContainer()
		return nil
	}, nil
}
