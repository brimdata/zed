package agg

import (
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type And struct {
	val *bool
}

func (a *And) Consume(v zng.Value) error {
	if v.Bytes == nil {
		return nil
	}
	if v.Type != zng.TypeBool {
		//l.TypeMismatch++
		return nil
	}
	if a.val == nil {
		b := true
		a.val = &b
	}
	*a.val = *a.val && zng.IsTrue(v.Bytes)
	return nil
}

func (a *And) Result(*zson.Context) (zng.Value, error) {
	if a.val == nil {
		return zng.Value{Type: zng.TypeBool}, nil
	}
	if *a.val {
		return zng.True, nil
	}
	return zng.False, nil
}

func (a *And) ConsumeAsPartial(v zng.Value) error {
	return a.Consume(v)
}

func (a *And) ResultAsPartial(*zson.Context) (zng.Value, error) {
	return a.Result(nil)
}

type Or struct {
	val *bool
}

func (o *Or) Consume(v zng.Value) error {
	if v.Bytes == nil {
		return nil
	}
	if v.Type != zng.TypeBool {
		//l.TypeMismatch++
		return nil
	}
	if o.val == nil {
		b := false
		o.val = &b
	}
	*o.val = *o.val || zng.IsTrue(v.Bytes)
	return nil
}

func (o *Or) Result(*zson.Context) (zng.Value, error) {
	if o.val == nil {
		return zng.Value{Type: zng.TypeBool}, nil
	}
	if *o.val {
		return zng.True, nil
	}
	return zng.False, nil
}

func (o *Or) ConsumeAsPartial(v zng.Value) error {
	return o.Consume(v)
}

func (o *Or) ResultAsPartial(*zson.Context) (zng.Value, error) {
	return o.Result(nil)
}
