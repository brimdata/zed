package index

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
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
	//XXX right now we use cutter but we might just keep the list of values?
	// If the key isn't present, the cutter will return error missing values.
	// I think we shouldn't error but return these when searching for null?
	// Or why not search for missing?
	return k.cutter.Eval(ectx, val)
}
