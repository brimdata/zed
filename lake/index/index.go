package index

import (
	"fmt"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Rule interface {
	RuleName() string
	RuleID() ksuid.KSUID
	RuleKeys() field.List
	Zed() string
	String() string
}

//XXX move Framesize into lake config.

// Index contains the runtime configuration for an index.
//XXX have rules with multiple fields, or multiple rules with one field each?
type FieldRule struct {
	ID     ksuid.KSUID `zng:"id"`
	Name   string      `zng:"name"`
	Fields field.List  `zng:"fields,omitempty"`
}

type TypeRule struct {
	ID    ksuid.KSUID `zng:"id"`
	Name  string      `zng:"name"`
	Value string      `zng:"type"`
}

type ZedRule struct {
	ID    ksuid.KSUID `zng:"id"`
	Name  string      `zng:"name"`
	Value string      `zng:"type"`
	Keys  field.List  `zng:"keys,omitempty"`
}

func ParseRule(name, pattern string) (Rule, error) {
	if pattern[0] == ':' {
		typ, err := zson.ParseType(zson.NewContext(), pattern[1:])
		if err != nil {
			return nil, err
		}
		return NewTypeRule(name, typ), nil
	}
	return NewFieldRule(name, pattern), nil
}

func NewTypeRule(name string, typ zng.Type) *TypeRule {
	return &TypeRule{
		Name: name,
		ID:   ksuid.New(),
		//XXX should store type as Zed type-value
		Value: tzngio.TypeString(typ),
	}
}

// NewFieldIndex creates an index that will index the field passed in as argument.
// It is currently an error to try to index a field name that appears as different types.
func NewFieldRule(name, keys string) *FieldRule {
	fields := field.DottedList(keys)
	if len(fields) != 1 {
		//XXX fix this
		panic("NewFieldRule: only one key supported")
	}
	return &FieldRule{
		Name:   name,
		ID:     ksuid.New(),
		Fields: fields,
	}
}

func NewZedRule(name, prog string, keys field.List) (*ZedRule, error) {
	// make sure it compiles
	if _, err := compiler.ParseProc(prog); err != nil {
		return nil, err
	}
	return &ZedRule{
		Name:  name,
		ID:    ksuid.New(),
		Keys:  keys,
		Value: prog,
	}, nil
}

// Equivalent determine if the provided Index is equivalent to the receiver. It
// should used to check if a Definition already contains and equivalent index.
/*
func (i Index) Equivalent(r2 Index) bool {
	if i.Kind != r2.Kind || i.Value != r2.Value {
		return false
	}
	if i.Kind == IndexZed {
		return i.Name == r2.Name
	}
	return true
}
*/

//XXX get rid of this
const keyName = "key"

func (f *FieldRule) Zed() string {
	if len(f.Fields) != 1 {
		//XXX multiple field keys not supported
		// The code below does a cut assignment presuming one key.
		// This is problematic.  We should change the index files
		// to presume the original names of the keys and just do
		// a non-assignmnet cut on all of the fields.
		panic("")
	}
	return fmt.Sprintf("cut %s:=%s | count() by %s | sort %s", keyName, f.Fields[0], keyName, keyName)
}

func (t *TypeRule) Zed() string {
	return fmt.Sprintf("explode this by %s as %s | count() by %s | sort %s", t.Value, keyName, keyName, keyName)
}

func (z *ZedRule) Zed() string {
	return z.Value
}

//XXX these don't make sense

func (f *FieldRule) String() string {
	return fmt.Sprintf("field->%s", f.Fields)
}

func (t *TypeRule) String() string {
	return fmt.Sprintf("type->%s", t.Value)
}

func (z *ZedRule) String() string {
	return fmt.Sprintf("zed->%s", z.Value)
}

func (f *FieldRule) RuleName() string {
	return f.Name
}

func (t *TypeRule) RuleName() string {
	return t.Name
}

func (z *ZedRule) RuleName() string {
	return z.Name
}

func (f *FieldRule) RuleID() ksuid.KSUID {
	return f.ID
}

func (t *TypeRule) RuleID() ksuid.KSUID {
	return t.ID
}

func (z *ZedRule) RuleID() ksuid.KSUID {
	return z.ID
}

func (f *FieldRule) RuleKeys() field.List {
	//fmt.Println("FIELD RULE RuleKeys", f.Fields)
	return field.DottedList(keyName)
}

func (t *TypeRule) RuleKeys() field.List {
	return nil
}

func (z *ZedRule) RuleKeys() field.List {
	return z.Keys
}
