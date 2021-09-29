package agg

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

type And struct {
	val *bool
}

func (a *And) Consume(v zed.Value) error {
	if v.Bytes == nil {
		return nil
	}
	if v.Type != zed.TypeBool {
		//l.TypeMismatch++
		return nil
	}
	if a.val == nil {
		b := true
		a.val = &b
	}
	*a.val = *a.val && zed.IsTrue(v.Bytes)
	return nil
}

func (a *And) Result(*zson.Context) (zed.Value, error) {
	if a.val == nil {
		return zed.Value{Type: zed.TypeBool}, nil
	}
	if *a.val {
		return zed.True, nil
	}
	return zed.False, nil
}

func (a *And) ConsumeAsPartial(v zed.Value) error {
	return a.Consume(v)
}

func (a *And) ResultAsPartial(*zson.Context) (zed.Value, error) {
	return a.Result(nil)
}

type Or struct {
	val *bool
}

func (o *Or) Consume(v zed.Value) error {
	if v.Bytes == nil {
		return nil
	}
	if v.Type != zed.TypeBool {
		//l.TypeMismatch++
		return nil
	}
	if o.val == nil {
		b := false
		o.val = &b
	}
	*o.val = *o.val || zed.IsTrue(v.Bytes)
	return nil
}

func (o *Or) Result(*zson.Context) (zed.Value, error) {
	if o.val == nil {
		return zed.Value{Type: zed.TypeBool}, nil
	}
	if *o.val {
		return zed.True, nil
	}
	return zed.False, nil
}

func (o *Or) ConsumeAsPartial(v zed.Value) error {
	return o.Consume(v)
}

func (o *Or) ResultAsPartial(*zson.Context) (zed.Value, error) {
	return o.Result(nil)
}
