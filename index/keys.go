package index

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
)

type Keyer struct {
	keys   []field.Path
	cutter *expr.Cutter
}

func NewKeyer(zctx *zed.Context, keys []field.Path) (*Keyer, error) {
	fields, resolvers := expr.NewAssignments(zctx, keys, keys)
	return &Keyer{
		keys:   keys,
		cutter: expr.NewCutter(zctx, fields, resolvers),
	}, nil
}

func (k *Keyer) Keys() []field.Path {
	return k.keys
}

func (k *Keyer) valueOfKeys(ectx expr.Context, val *zed.Value) *zed.Value {
	return k.cutter.Eval(ectx, val)
}
