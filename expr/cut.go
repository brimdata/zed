package expr

import (
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type CutFunc struct {
	*Cutter
}

func NewCutFunc(zctx *resolver.Context, fieldRefs []field.Static, fieldExprs []Evaluator) (*CutFunc, error) {
	c, err := NewCutter(zctx, fieldRefs, fieldExprs)
	if err != nil {
		return nil, err
	}
	return &CutFunc{c}, nil
}

func (c *CutFunc) Eval(rec *zng.Record) (zng.Value, error) {
	out, err := c.Apply(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if out == nil {
		return zng.Value{}, ErrNoSuchField
	}
	return zng.Value{out.Type, out.Raw}, nil
}
