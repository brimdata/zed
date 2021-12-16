package agg

import (
	"github.com/brimdata/zed"
)

type And struct {
	val *bool
}

var _ Function = (*And)(nil)

func (a *And) Consume(val *zed.Value) {
	if val.IsNull() || zed.AliasOf(val.Type) != zed.TypeBool {
		return
	}
	if a.val == nil {
		b := true
		a.val = &b
	}
	*a.val = *a.val && zed.IsTrue(val.Bytes)
}

func (a *And) Result(*zed.Context) *zed.Value {
	if a.val == nil {
		return zed.NullBool
	}
	if *a.val {
		return zed.True
	}
	return zed.False
}

func (a *And) ConsumeAsPartial(val *zed.Value) {
	if val.Type != zed.TypeBool {
		panic("and: partial not a bool")
	}
	a.Consume(val)
}

func (a *And) ResultAsPartial(*zed.Context) *zed.Value {
	return a.Result(nil)
}

type Or struct {
	val *bool
}

var _ Function = (*Or)(nil)

func (o *Or) Consume(val *zed.Value) {
	if val.IsNull() || zed.AliasOf(val.Type) != zed.TypeBool {
		return
	}
	if o.val == nil {
		b := false
		o.val = &b
	}
	*o.val = *o.val || zed.IsTrue(val.Bytes)
}

func (o *Or) Result(*zed.Context) *zed.Value {
	if o.val == nil {
		return zed.NullBool
	}
	if *o.val {
		return zed.True
	}
	return zed.False
}

func (o *Or) ConsumeAsPartial(val *zed.Value) {
	if val.Type != zed.TypeBool {
		panic("or: partial not a bool")
	}
	o.Consume(val)
}

func (o *Or) ResultAsPartial(*zed.Context) *zed.Value {
	return o.Result(nil)
}
