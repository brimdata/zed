package index

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/pkg/field"
)

type Keyer struct {
	keys   []field.Path
	cutter *expr.Cutter
}

func NewKeyer(zctx *zed.Context, keys []field.Path) (*Keyer, error) {
	fields, resolvers := compiler.CompileAssignments(zctx, keys, keys)
	cutter, err := expr.NewCutter(zctx, fields, resolvers)
	if err != nil {
		return nil, err
	}
	return &Keyer{
		keys:   keys,
		cutter: cutter,
	}, nil
}

func (k *Keyer) Keys() []field.Path {
	return k.keys
}

func (k *Keyer) valueOfKeys(ectx expr.Context, val *zed.Value) *zed.Value {
	return k.cutter.Eval(ectx, val)
}
