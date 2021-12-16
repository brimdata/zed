package agg

import (
	"github.com/brimdata/zed"
)

type And struct {
	val *bool
}

var _ Function = (*And)(nil)

func (a *And) Consume(v zed.Value) {
	if v.Bytes == nil {
		return
	}
	if v.Type != zed.TypeBool {
		//XXX coerce?
		return
	}
	if a.val == nil {
		b := true
		a.val = &b
	}
	*a.val = *a.val && zed.IsTrue(v.Bytes)
}

func (a *And) Result(*zed.Context) zed.Value {
	if a.val == nil {
		return zed.Value{Type: zed.TypeBool}
	}
	if *a.val {
		return zed.True
	}
	return zed.False
}

func (a *And) ConsumeAsPartial(v zed.Value) {
	//XXX check type and panic?
	a.Consume(v)
}

func (a *And) ResultAsPartial(*zed.Context) zed.Value {
	return a.Result(nil)
}

type Or struct {
	val *bool
}

var _ Function = (*Or)(nil)

func (o *Or) Consume(v zed.Value) {
	if v.Bytes == nil {
		return
	}
	if v.Type != zed.TypeBool {
		//XXX coerce?
		return
	}
	if o.val == nil {
		b := false
		o.val = &b
	}
	*o.val = *o.val || zed.IsTrue(v.Bytes)
}

func (o *Or) Result(*zed.Context) zed.Value {
	if o.val == nil {
		return zed.Value{Type: zed.TypeBool}
	}
	if *o.val {
		return zed.True
	}
	return zed.False
}

func (o *Or) ConsumeAsPartial(v zed.Value) {
	o.Consume(v)
}

func (o *Or) ResultAsPartial(*zed.Context) zed.Value {
	return o.Result(nil)
}
