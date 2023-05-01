package vam

import (
	"fmt"
	"strings"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"golang.org/x/sync/errgroup"
)

type Projection struct {
	object     *vcache.Object
	typeKeys   []int32
	projectors []projector
	builder    zcode.Builder
	off        int
}

// One projector per top-level type
type projector struct {
	recType *zed.TypeRecord
	build   builder
}

var _ zio.Reader = (*Projection)(nil)

func NewProjection(o *vcache.Object, names []string) (*Projection, error) {
	//XXX just handles top-level names for now, fix this to build records
	// with "as field.Path"... use compiler to compile a cut proc.
	var group errgroup.Group
	projectors := make([]projector, len(o.Types()))
	//XXX need concurrency over the typekeys too
	for typeKey := range o.Types() {
		typeKey := uint32(typeKey)
		builders := make([]builder, len(names))
		types := make([]zed.Type, len(names))
		var mu sync.Mutex
		for pos, name := range names {
			pos := pos
			name := name
			group.Go(func() error {
				vec, err := o.Load(typeKey, []string{name}) //XXX need full path
				if err != nil {
					return err
				}
				builder, err := newBuilder(vec)
				if err != nil {
					return err
				}
				mu.Lock()
				types[pos] = vec.Type()
				builders[pos] = builder
				mu.Unlock()
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return nil, err
		}
		var packed []builder
		var fields []zed.Field
		for k, b := range builders {
			if b != nil {
				fields = append(fields, zed.Field{Type: types[k], Name: names[k]})
				packed = append(packed, b)
			}
		}
		if len(packed) == 0 {
			continue
		}
		recType, err := o.LocalContext().LookupTypeRecord(fields)
		if err != nil {
			return nil, err
		}
		projectors[typeKey] = projector{
			recType: recType,
			build: func(b *zcode.Builder) bool {
				b.BeginContainer()
				for _, materialize := range packed {
					if ok := materialize(b); !ok {
						return false
					}
				}
				b.EndContainer()
				return true
			},
		}
	}
	empty := true
	for k := 0; k < len(projectors); k++ {
		if projectors[k].build != nil {
			empty = false
			break
		}
	}
	if empty {
		return nil, fmt.Errorf("none of the specified fields were found: %s", strings.Join(names, ", "))
	}
	return &Projection{
		object:     o,
		typeKeys:   o.TypeKeys(),
		projectors: projectors,
	}, nil
}

// XXX Let's use Pull() here... read whole column into Batch for better perf
func (p *Projection) Read() (*zed.Value, error) {
	for {
		if p.off >= len(p.typeKeys) {
			return nil, nil
		}
		typeKey := p.typeKeys[p.off]
		p.off++
		projector := p.projectors[typeKey]
		if projector.build != nil {
			p.builder.Truncate()
			if ok := projector.build(&p.builder); !ok {
				return nil, nil
			}
			return zed.NewValue(projector.recType, p.builder.Bytes().Body()), nil
		}
	}
}
