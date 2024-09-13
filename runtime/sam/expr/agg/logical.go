package agg

import (
	"github.com/brimdata/zed"
)

type And struct {
	val *bool
}

var _ Function = (*And)(nil)

func (a *And) Consume(val zed.Value) {
	if val.IsNull() || zed.TypeUnder(val.Type()) != zed.TypeBool {
		return
	}
	if a.val == nil {
		b := true
		a.val = &b
	}
	*a.val = *a.val && val.Bool()
}

func (a *And) Result(*zed.Context, *zed.Arena) zed.Value {
	if a.val == nil {
		return zed.NullBool
	}
	return zed.NewBool(*a.val)
}

func (a *And) ConsumeAsPartial(val zed.Value) {
	if val.Type() != zed.TypeBool {
		panic("and: partial not a bool")
	}
	a.Consume(val)
}

func (a *And) ResultAsPartial(*zed.Context, *zed.Arena) zed.Value {
	return a.Result(nil, nil)
}

type Or struct {
	val *bool
}

var _ Function = (*Or)(nil)

func (o *Or) Consume(val zed.Value) {
	if val.IsNull() || zed.TypeUnder(val.Type()) != zed.TypeBool {
		return
	}
	if o.val == nil {
		b := false
		o.val = &b
	}
	*o.val = *o.val || val.Bool()
}

func (o *Or) Result(*zed.Context, *zed.Arena) zed.Value {
	if o.val == nil {
		return zed.NullBool
	}
	return zed.NewBool(*o.val)
}

func (o *Or) ConsumeAsPartial(val zed.Value) {
	if val.Type() != zed.TypeBool {
		panic("or: partial not a bool")
	}
	o.Consume(val)
}

func (o *Or) ResultAsPartial(*zed.Context, *zed.Arena) zed.Value {
	return o.Result(nil, nil)
}
