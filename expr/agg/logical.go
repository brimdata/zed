package agg

import (
	"github.com/brimdata/zed"
)

type And struct {
	val *bool
}

var _ Function = (*And)(nil)

func (a *And) Consume(val *zed.Value) {
	if val.IsNull() {
		return
	}
	if val.Type != zed.TypeBool {
		//XXX coerce?
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
		//XXX singleton
		return &zed.Value{Type: zed.TypeBool}
	}
	if *a.val {
		return zed.True
	}
	return zed.False
}

func (a *And) ConsumeAsPartial(val *zed.Value) {
	//XXX check type and panic?
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
	if val.Bytes == nil {
		return
	}
	if val.Type != zed.TypeBool {
		//XXX coerce?
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
		//XXX singleton
		return &zed.Value{Type: zed.TypeBool}
	}
	if *o.val {
		return zed.True
	}
	return zed.False
}

func (o *Or) ConsumeAsPartial(val *zed.Value) {
	o.Consume(val)
}

func (o *Or) ResultAsPartial(*zed.Context) *zed.Value {
	return o.Result(nil)
}
