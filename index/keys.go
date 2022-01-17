package index

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zson"
)

type Keyer struct {
	keys   []field.Path
	cutter *expr.Cutter
	valid  map[zed.Type]struct{}
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
		valid:  make(map[zed.Type]struct{}),
	}, nil
}

func (k *Keyer) Keys() []field.Path {
	return k.keys
}

func (k *Keyer) KeyOf(ectx expr.Context, val *zed.Value) (*zed.Value, error) {
	key := k.cutter.Eval(ectx, val)
	if _, ok := k.valid[key.Type]; ok {
		return key, nil
	}
	recType, ok := key.Type.(*zed.TypeRecord)
	if !ok {
		return nil, fmt.Errorf("index key is not a record: %s", zson.MustFormatValue(*val))
	}
	if _, ok := k.valid[key.Type]; !ok {
		for _, col := range recType.Columns {
			if _, ok := zed.TypeUnder(col.Type).(*zed.TypeError); ok {
				return nil, fmt.Errorf("no index key field %q present in record: %s", col.Name, zson.MustFormatValue(*val))
			}
		}
		k.valid[key.Type] = struct{}{}
	}
	return key, nil
}
