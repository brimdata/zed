package expr

import (
	"github.com/brimsec/zq/zng"
)

type cutFunc struct {
	*Cutter
}

func NewCutFunc(c *Cutter) *cutFunc {
	return &cutFunc{c}
}

func (c *cutFunc) Eval(rec *zng.Record) (zng.Value, error) {
	out, err := c.Cut(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if out == nil {
		return zng.Value{}, ErrNoSuchField
	}
	return zng.Value{out.Type, out.Raw}, nil
}
