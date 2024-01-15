package vcache

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	"github.com/brimdata/zed/zson"
)

func (l *loader) loadUnion(any *vector.Any, typ *zed.TypeUnion, path field.Path, m *vng.Union) (*vector.Union, error) {
	if *any == nil {
		*any = vector.NewUnion(typ)
	}
	vec, ok := (*any).(*vector.Union)
	if !ok {
		return nil, fmt.Errorf("system error: vcache.loadUnion not a union type %q", zson.String(vec.Typ))
	}
	tags, err := vng.ReadIntVector(m.Tags, l.r)
	if err != nil {
		return nil, err
	}
	vec.Tags = tags
	//XXX should just load paths we want here?  for now, load everything.
	for k := range vec.Values {
		var err error
		_, err = l.loadVector(&vec.Values[k], typ.Types[k], path, m.Values[k])
		if err != nil {
			return nil, err
		}
	}
	return vec, nil
}

/*
func (u *Union) NewIter(reader io.ReaderAt) (iterator, error) {
	if u.tags == nil {
		tags, err := vng.ReadIntVector(u.segmap, reader)
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
			return fmt.Errorf("VNG cache: bad union tag encountered %d of %d", tag, len(iters))
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
*/
