package index

import (
	"bytes"
	"fmt"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type RuleKind string

const (
	RuleType  RuleKind = "type"
	RuleField RuleKind = "field"
	RuleZed   RuleKind = "zed"
)

// Rule contains the runtime configuration for an indexing rule.
type Rule struct {
	Framesize int            `zng:"framesize,omitempty"`
	ID        ksuid.KSUID    `zng:"id"`
	Name      string         `zng:"name,omitempty"`
	Keys      []field.Static `zng:"keys,omitempty"`
	Kind      RuleKind       `zng:"kind"`
	Value     string         `zng:"type"`
}

func ParseRule(pattern string) (Rule, error) {
	if pattern[0] == ':' {
		typ, err := zson.NewContext().LookupByName(pattern[1:])
		if err != nil {
			return Rule{}, err
		}
		return NewTypeRule(typ), nil
	}
	return NewFieldRule(pattern), nil
}

func NewTypeRule(typ zng.Type) Rule {
	return Rule{
		ID:    ksuid.New(),
		Kind:  RuleType,
		Value: tzngio.TypeString(typ),
	}
}

// NewFieldRule creates an indexing rule that will index the field passed in as argument.
// It is currently an error to try to index a field name that appears as different types.
func NewFieldRule(fieldName string) Rule {
	return Rule{
		ID:    ksuid.New(),
		Kind:  RuleField,
		Value: fieldName,
	}
}

func UnmarshalRule(b []byte) (Rule, error) {
	zctx := zson.NewContext()
	zr := zngio.NewReader(bytes.NewReader(b), zctx)
	rec, err := zr.Read()
	if err != nil {
		return Rule{}, err
	}
	r := Rule{}
	return r, resolver.UnmarshalRecord(rec, &r)
}

func NewZedRule(prog, name string, keys []field.Static) (Rule, error) {
	// make sure it compiles
	if _, err := compiler.ParseProc(prog); err != nil {
		return Rule{}, err
	}
	return Rule{
		ID:    ksuid.New(),
		Keys:  keys,
		Kind:  RuleZed,
		Name:  name,
		Value: prog,
	}, nil
}

// Equivalent determine if the provided Rule is equivalent to the receiver. It
// should used to check if a Definition already contains and equivalent rule.
func (r Rule) Equivalent(r2 Rule) bool {
	if r.Kind != r2.Kind || r.Value != r2.Value {
		return false
	}
	if r.Kind == RuleZed {
		return r.Name == r2.Name
	}
	return true
}

func (r Rule) Proc() (ast.Proc, error) {
	switch r.Kind {
	case RuleType:
		return r.typeProc()
	case RuleField:
		return r.fieldProc()
	case RuleZed:
		return r.zqlProc()
	default:
		return nil, fmt.Errorf("unknown rule kind: %s", r.Kind)
	}
}

var keyName = field.New("key")

var keyAst = ast.Assignment{
	LHS: ast.NewDotExpr(keyName),
	RHS: ast.NewDotExpr(keyName),
}
var countAst = ast.NewAggAssignment("count", nil, nil)

// NewFieldRule creates an indexing rule that will index all fields of
// the type passed in as argument.
func (r Rule) typeProc() (ast.Proc, error) {
	return &ast.Sequential{
		Kind: "Sequential",
		Procs: []ast.Proc{
			&ast.TypeSplitter{
				Key:      keyName,
				TypeName: r.Value,
			},
			&ast.Summarize{
				Kind: "Summarize",
				Keys: []ast.Assignment{keyAst},
				Aggs: []ast.Assignment{countAst},
			},
			&ast.Sort{
				Kind: "Sort",
				Args: []ast.Expr{ast.NewDotExpr(keyName)},
			},
		},
	}, nil
}

func (r Rule) fieldProc() (ast.Proc, error) {
	return &ast.Sequential{
		Kind: "Sequential",
		Procs: []ast.Proc{
			&ast.FieldCutter{
				Field: field.Dotted(r.Value),
				Out:   keyName,
			},
			&ast.Summarize{
				Kind: "Summarize",
				Keys: []ast.Assignment{keyAst},
				Aggs: []ast.Assignment{countAst},
			},
			&ast.Sort{
				Kind: "Sort",
				Args: []ast.Expr{ast.NewDotExpr(keyName)},
			},
		},
	}, nil
}

func (r Rule) zqlProc() (ast.Proc, error) {
	return compiler.ParseProc(r.Value)
}

func (r Rule) String() string {
	name := r.Value
	if r.Kind == RuleZed {
		name = r.Name
	}
	return fmt.Sprintf("%s->%s", r.Kind, name)
}

type Rules []Rule

func (rules Rules) Lookup(id ksuid.KSUID) *Rule {
	if i := rules.indexOf(id); i >= 0 {
		return &rules[i]
	}
	return nil
}

// Add checks if Rules already has an equivalent Rule and if it does not
// returns Rules with the Rule appended to it. Returns a non-nil Rule pointer if
// an equivalent Rule is found.
func (rules Rules) Add(rule Rule) (Rules, *Rule) {
	for _, r := range rules {
		if r.Equivalent(rule) {
			return rules, &r
		}
	}
	return append(rules, rule), nil
}

// LookupDelete checks the Rules list for a rule matching the provided ID and
// returns the deleted Rule if found.
func (rules Rules) LookupDelete(id ksuid.KSUID) (Rules, *Rule) {
	if i := rules.indexOf(id); i >= 0 {
		rule := rules[i]
		return append(rules[:i], rules[i+1:]...), &rule
	}
	return rules, nil
}

func (rules Rules) indexOf(id ksuid.KSUID) int {
	for i, rule := range rules {
		if rule.ID == id {
			return i
		}
	}
	return -1
}

// func (rules) Rules
