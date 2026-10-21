package loader

import (
	"sync"

	"github.com/superdb/super/bsup"
	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/vector"
)

type fusion struct {
	mu       sync.Mutex
	bctx     *bsup.Context
	meta     *bsup.Fusion
	len      uint32
	values   shadow
	subtypes *typevalue
}

func newFusion(bctx *bsup.Context, meta *bsup.Fusion) *fusion {
	return &fusion{
		bctx: bctx,
		meta: meta,
		len:  meta.Len(bctx),
	}
}

func (f *fusion) length() uint32 {
	return f.len
}

func (f *fusion) unmarshal(bctx *bsup.Context, projection field.Projection) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.values == nil {
		f.values = newShadow(bctx, f.meta.Values)
	}
	if f.subtypes == nil {
		f.subtypes = newTypeValue(bctx, bctx.Lookup(f.meta.Subtypes).(*bsup.TypeValue))
	}
	f.values.unmarshal(bctx, projection)
}

func (f *fusion) project(loader *FrameLoader, projection field.Projection) vector.Any {
	vec := f.values.project(loader, projection)
	if len(projection) != 0 {
		return vec
	}
	typ := loader.sctx.MustLookupTypeFusion(vec.Type())
	return vector.NewFusionWithLoader(loader.sctx, typ, f.subtypes.newLoader(loader), vec)
}
