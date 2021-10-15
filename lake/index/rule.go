package index

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Rule interface {
	CreateTime() nano.Ts
	RuleName() string
	RuleID() ksuid.KSUID
	RuleKeys() field.List
	Zed() string
	String() string
}

type FieldRule struct {
	Ts     nano.Ts     `zed:"ts"`
	ID     ksuid.KSUID `zed:"id"`
	Name   string      `zed:"name"`
	Fields field.List  `zed:"fields,omitempty"`
}

type TypeRule struct {
	Ts   nano.Ts     `zed:"ts"`
	ID   ksuid.KSUID `zed:"id"`
	Name string      `zed:"name"`
	Type string      `zed:"type"`
}

type AggRule struct {
	Ts     nano.Ts     `zed:"ts"`
	ID     ksuid.KSUID `zed:"id"`
	Name   string      `zed:"name"`
	Script string      `zed:"script"`
}

func NewFieldRule(name, keys string) *FieldRule {
	fields := field.DottedList(keys)
	if len(fields) != 1 {
		//XXX fix this
		panic("NewFieldRule: only one key supported")
	}
	return &FieldRule{
		Ts:     nano.Now(),
		Name:   name,
		ID:     ksuid.New(),
		Fields: fields,
	}
}

func NewTypeRule(name string, typ zed.Type) *TypeRule {
	return &TypeRule{
		Ts:   nano.Now(),
		Name: name,
		ID:   ksuid.New(),
		Type: zson.FormatType(typ),
	}
}

func NewAggRule(name, prog string) (*AggRule, error) {
	// make sure it compiles
	if _, err := compiler.ParseProc(prog); err != nil {
		return nil, err
	}
	return &AggRule{
		Ts:     nano.Now(),
		Name:   name,
		ID:     ksuid.New(),
		Script: prog,
	}, nil
}

// Equivalent returns true if the two rules create the same index object.
func Equivalent(a, b Rule) bool {
	switch ra := a.(type) {
	case *FieldRule:
		if rb, ok := b.(*FieldRule); ok {
			return ra.Fields.Equal(rb.Fields)
		}
	case *TypeRule:
		if rb, ok := b.(*TypeRule); ok {
			return ra.Type == rb.Type
		}
	case *AggRule:
		if rb, ok := b.(*AggRule); ok {
			return ra.Script == rb.Script
		}
	}
	return false
}

func (f *FieldRule) Zed() string {
	var fields string
	for i, field := range f.Fields {
		if i > 0 {
			fields += ", "
		}
		fields += field.String()
	}
	return fmt.Sprintf("cut %s | count() by %s | sort %s", fields, fields, fields)
}

func (t *TypeRule) Zed() string {
	// XXX See issue #3140 as this does not allow for multiple type keys
	return fmt.Sprintf("explode this by %s as key | count() by key | sort key", t.Type)
}

func (a *AggRule) Zed() string {
	return a.Script
}

func (f *FieldRule) String() string {
	return fmt.Sprintf("rule %s field %s", f.ID, f.Fields)
}

func (t *TypeRule) String() string {
	return fmt.Sprintf("rule %s type %s", t.ID, t.Type)
}

func (a *AggRule) String() string {
	return fmt.Sprintf("rule %s agg %q", a.ID, a.Script)
}

func (f *FieldRule) CreateTime() nano.Ts {
	return f.Ts
}

func (t *TypeRule) CreateTime() nano.Ts {
	return t.Ts
}

func (a *AggRule) CreateTime() nano.Ts {
	return a.Ts
}

func (f *FieldRule) RuleName() string {
	return f.Name
}

func (t *TypeRule) RuleName() string {
	return t.Name
}

func (a *AggRule) RuleName() string {
	return a.Name
}

func (f *FieldRule) RuleID() ksuid.KSUID {
	return f.ID
}

func (t *TypeRule) RuleID() ksuid.KSUID {
	return t.ID
}

func (a *AggRule) RuleID() ksuid.KSUID {
	return a.ID
}

func (f *FieldRule) RuleKeys() field.List {
	return f.Fields
}

func (t *TypeRule) RuleKeys() field.List {
	return field.DottedList("key")
}

func (a *AggRule) RuleKeys() field.List {
	// XXX can get these by analyzing the compiled script
	return nil
}

// newLookupKey creates a Zed Record that can be used as a lookup key for an
// index created for the provided Rule. The Values provided must be in order
// with the Key in the rule that it will be paired with it.
func newLookupKey(zctx *zed.Context, r Rule, values []zed.Value) (*zed.Value, error) {
	keys := r.RuleKeys()
	// XXX Ensure length of values equals the length of Keys in the Rule or
	// else zed.ColumnBuilder will throw an error on Encode. We should adjust
	// zed.ColumnBuilder to be less strict.
	if n := len(keys) - len(values); n < 0 {
		values = values[:len(keys)]
	} else if n > 0 {
		values = append(values, make([]zed.Value, n)...)
	}
	builder, err := zed.NewColumnBuilder(zctx, keys)
	if err != nil {
		return nil, err
	}
	types := make([]zed.Type, len(values))
	for i, v := range values {
		types[i] = v.Type
		builder.Append(v.Bytes, zed.IsContainerType(v.Type))
	}
	b, err := builder.Encode()
	if err != nil {
		return nil, err
	}
	typ := zctx.MustLookupTypeRecord(builder.TypedColumns(types))
	return zed.NewValue(typ, b), nil
}
