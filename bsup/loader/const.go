package loader

import (
	"github.com/superdb/super"
	"github.com/superdb/super/bsup"
	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/vector"
)

type const_ struct {
	meta *bsup.Const
	len  uint32
}

func newConst(bctx *bsup.Context, meta *bsup.Const) *const_ {
	return &const_{meta: meta, len: meta.Len(bctx)}
}

func (c *const_) length() uint32 {
	return c.len
}

func (*const_) unmarshal(*bsup.Context, field.Projection) {}

func (c *const_) project(loader *FrameLoader, projection field.Projection) vector.Any {
	// Map the const super.Value in the bsup's type context to
	// a new one in the query type context.
	val := c.meta.Value
	if val.IsNull() {
		return vector.NewNull(c.length())
	}
	typ, err := loader.sctx.TranslateType(val.Type())
	if err != nil {
		panic(err)
	}
	return vector.NewConstFromValue(loader.sctx, super.NewValue(typ, val.Bytes()), c.length())
}
func (k *const_) strings(c *collector) {
	val := k.meta.Value
	if val.IsNull() {
		return
	}
	typ := super.TypeUnder(val.Type())
	if typ.Kind() == super.PrimitiveKind && typ.ID() != super.IDString {
		return
	}
	// Complex consts: every embedded string appears verbatim in the
	// encoding, so the raw bytes are a safe superset to scan.
	c.append(val.Bytes())
}
