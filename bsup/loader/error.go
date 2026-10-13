package loader

import (
	"sync"

	"github.com/superdb/super/bsup"
	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/vector"
)

type error_ struct {
	mu     sync.Mutex
	meta   *bsup.Error
	len    uint32
	values shadow
}

func newError(bctx *bsup.Context, meta *bsup.Error) *error_ {
	return &error_{meta: meta, len: meta.Len(bctx)}
}

func (e *error_) length() uint32 {
	return e.len
}

func (e *error_) unmarshal(bctx *bsup.Context, projection field.Projection) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.values == nil {
		e.values = newShadow(bctx, e.meta.Values)
	}
	e.values.unmarshal(bctx, projection)
}

func (e *error_) project(loader *FrameLoader, projection field.Projection) vector.Any {
	vec := e.values.project(loader, projection)
	typ := loader.sctx.LookupTypeError(vec.Type())
	return vector.NewError(typ, vec)
}
