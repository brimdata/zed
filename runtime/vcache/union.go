package vcache

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zst"
	"github.com/brimdata/zed/zst/vector"
	"golang.org/x/sync/errgroup"
)

type Union struct {
	values []Vector
	tags   []int32
	segmap []vector.Segment
}

func NewUnion(union *vector.Union, r io.ReaderAt) (*Union, error) {
	values := make([]Vector, 0, len(union.Values))
	for _, val := range union.Values {
		v, err := NewVector(val, r)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return &Union{
		values: values,
		segmap: union.Tags,
	}, nil
}

func (u *Union) NewIter(reader io.ReaderAt) (iterator, error) {
	if u.tags == nil {
		tags, err := zst.ReadIntVector(u.segmap, reader)
		if err != nil {
			return nil, err
		}
		u.tags = tags
	}
	var group errgroup.Group
	iters := make([]iterator, len(u.values))
	for k, v := range u.values {
		which := k
		vals := v
		group.Go(func() error {
			it, err := vals.NewIter(reader)
			if err != nil {
				return err
			}
			iters[which] = it
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	off := 0
	return func(b *zcode.Builder) error {
		tag := u.tags[off]
		off++
		if tag < 0 || int(tag) >= len(iters) {
			return fmt.Errorf("zst cache: bad union tag encountered %d of %d", tag, len(iters))
		}
		b.BeginContainer()
		b.Append(zed.EncodeInt(int64(tag)))
		if err := iters[tag](b); err != nil {
			return err
		}
		b.EndContainer()
		return nil
	}, nil
}
