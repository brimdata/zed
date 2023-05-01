package vam

import (
	"fmt"
	"strings"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/vector"
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
	build   vector.Builder
}

var _ zio.Reader = (*Projection)(nil)

func NewProjection(o *vcache.Object, names []string) (*Projection, error) {
	//XXX just handles top-level names for now, fix this to build records
	// with "as field.Path"... use compiler to compile a cut proc.
	var group errgroup.Group
	projectors := make([]projector, len(o.Types()))
	//XXX need concurrency over the typekeys too
	// For each type, we create a record vector comprising each of the fields
	// that are present in that type (or we skip the type if no such fields are
	// found).  When we encounter a partial projection, we fill the missing fields
	// with a const vector of error("missing").  Then, if we have no matching fields
	// in any of the types we return an error; if we have one matching type, we
	// use the builder on the corresponding record vector; if we have more than one,
	// we create a union vector and map the type keys from the vector object to
	// the tags of the computed union.
	for typeKey := range o.Types() {
		typeKey := uint32(typeKey)
		//XXX instead of doing this we should just make vector.Records that
		// represent the projection and call NewBuilder on that. We still have
		// to load the underlying fields.
		vecs := make([]vector.Any, len(names))
		var mu sync.Mutex
		for pos, name := range names {
			pos := pos
			name := name
			group.Go(func() error {
				vec, err := o.Load(typeKey, []string{name}) //XXX need full path
				if err != nil {
					return err
				}
				mu.Lock()
				vecs[pos] = vec
				mu.Unlock()
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return nil, err
		}
		var fields []zed.Field
		for k, vec := range vecs {
			if vec != nil {
				fields = append(fields, zed.Field{Type: vec.Type(), Name: names[k]})
			}
		}
		if len(fields) == 0 {
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
		object: o,
		/* XXX this is Jamie's UnionVector idea though
		it's not quite the same as a zed union */
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
