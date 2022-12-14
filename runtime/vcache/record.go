package vcache

import (
	"io"

	"github.com/brimdata/zed/vng/vector"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/sync/errgroup"
)

type Record []Vector

func NewRecord(fields []vector.Field, r io.ReaderAt) (Record, error) {
	record := make([]Vector, 0, len(fields))
	for _, field := range fields {
		v, err := NewVector(field.Values, r)
		if err != nil {
			return nil, err
		}
		record = append(record, v)
	}
	return record, nil
}

func (r Record) NewIter(reader io.ReaderAt) (iterator, error) {
	fields := make([]iterator, len(r))
	var group errgroup.Group
	for k, f := range r {
		which := k
		field := f
		group.Go(func() error {
			it, err := field.NewIter(reader)
			if err != nil {
				return err
			}
			fields[which] = it
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return func(b *zcode.Builder) error {
		b.BeginContainer()
		for _, it := range fields {
			if err := it(b); err != nil {
				return err
			}
		}
		b.EndContainer()
		return nil
	}, nil
}
