package vcache

import (
	"io"

	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
)

type Map struct {
	segmap  []vector.Segment
	keys    Vector
	values  Vector
	lengths []int32
}

func NewMap(m *vector.Map, r io.ReaderAt) (*Map, error) {
	keys, err := NewVector(m.Keys, r)
	if err != nil {
		return nil, err
	}
	values, err := NewVector(m.Values, r)
	if err != nil {
		return nil, err
	}
	return &Map{
		segmap: m.Lengths,
		keys:   keys,
		values: values,
	}, nil
}

func (m *Map) NewIter(reader io.ReaderAt) (iterator, error) {
	// The lengths vector is typically large and is loaded on demand.
	if m.lengths == nil {
		lengths, err := vng.ReadIntVector(m.segmap, reader)
		if err != nil {
			return nil, err
		}
		m.lengths = lengths
	}
	keys, err := m.keys.NewIter(reader)
	if err != nil {
		return nil, err
	}
	values, err := m.values.NewIter(reader)
	if err != nil {
		return nil, err
	}
	off := 0
	return func(b *zcode.Builder) error {
		len := m.lengths[off]
		off++
		b.BeginContainer()
		for ; len > 0; len-- {
			if err := keys(b); err != nil {
				return err
			}
			if err := values(b); err != nil {
				return err
			}
		}
		b.EndContainer()
		return nil
	}, nil
}
