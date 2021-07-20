package index

import (
	"fmt"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zng"
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
	Ts     nano.Ts     `zng:"ts"`
	ID     ksuid.KSUID `zng:"id"`
	Name   string      `zng:"name"`
	Fields field.List  `zng:"fields,omitempty"`
}

type TypeRule struct {
	Ts   nano.Ts     `zng:"ts"`
	ID   ksuid.KSUID `zng:"id"`
	Name string      `zng:"name"`
	Type string      `zng:"type"`
}

type AggRule struct {
	Ts     nano.Ts     `zng:"ts"`
	ID     ksuid.KSUID `zng:"id"`
	Name   string      `zng:"name"`
	Script string      `zng:"script"`
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

func NewTypeRule(name string, typ zng.Type) *TypeRule {
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

// XXX See issue #2923
const keyName = "key"

func (f *FieldRule) Zed() string {
	if len(f.Fields) != 1 {
		// XXX see issue #2923.  Multiple field keys not supported.
		// The code below does a cut assignment presuming one key.
		// This is problematic.  We should change the index files
		// to presume the original names of the keys and just do
		// a non-assignmnet cut on all of the fields.
		panic("")
	}
	return fmt.Sprintf("cut %s:=%s | count() by %s | sort %s", keyName, f.Fields[0], keyName, keyName)
}

func (t *TypeRule) Zed() string {
	// XXX See issue #2923 as this doesn't make sense.
	return fmt.Sprintf("explode this by %s as %s | count() by %s | sort %s", t.Type, keyName, keyName, keyName)
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
	return field.DottedList(keyName)
}

func (t *TypeRule) RuleKeys() field.List {
	return nil
}

func (a *AggRule) RuleKeys() field.List {
	// XXX can get these by analyzing the compiled script
	return nil
}
