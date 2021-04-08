package index

import (
	"bytes"
	"fmt"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/brimdata/zed/zson"
)

type RuleKind string

const (
	RuleType  RuleKind = "type"
	RuleField RuleKind = "field"
	RuleZQL   RuleKind = "zql"
)

// Rule contains the runtime configuration for an indexing rule.
type Rule struct {
	Kind      RuleKind       `zng:"kind"`
	Type      string         `zng:"type"`
	Name      string         `zng:"name"`
	Field     string         `zng:"field"`
	Framesize int            `zng:"framesize"`
	Input     string         `zng:"input_path"`
	Keys      []field.Static `zng:"keys"`
	ZQL       string         `zng:"zql"`
}

func NewRule(pattern string) (Rule, error) {
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
		Kind: RuleType,
		Type: tzngio.TypeString(typ),
	}
}

// NewFieldRule creates an indexing rule that will index the field passed in as argument.
// It is currently an error to try to index a field name that appears as different types.
func NewFieldRule(fieldName string) Rule {
	return Rule{
		Kind:  RuleField,
		Field: fieldName,
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

func NewZqlRule(prog, name string, keys []field.Static) (Rule, error) {
	// make sure it compiles
	if _, err := compiler.ParseProc(prog); err != nil {
		return Rule{}, err
	}
	return Rule{
		Kind: RuleZQL,
		ZQL:  prog,
		Name: name,
		Keys: keys,
	}, nil
}

// Equivalent determine if the provided Rule is equivalent to the receiver. It
// should used to check if a Definition already contains and equivalent rule.
func (r Rule) Equivalent(r2 Rule) bool {
	if r.Kind != r2.Kind || r.Input != r2.Input {
		return false
	}
	switch r.Kind {
	case RuleType:
		return r.Type == r2.Type
	case RuleField:
		return r.Field == r2.Field
	case RuleZQL:
		return r.Name == r2.Name && r.ZQL == r2.ZQL
	default:
		return false
	}
}

func (r Rule) Proc() (ast.Proc, error) {
	switch r.Kind {
	case RuleType:
		return r.typeProc()
	case RuleField:
		return r.fieldProc()
	case RuleZQL:
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
				TypeName: r.Type,
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
				Field: field.Dotted(r.Field),
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
	return compiler.ParseProc(r.ZQL)
}

func (r Rule) String() string {
	var name string
	switch r.Kind {
	case RuleType:
		name = r.Type
	case RuleField:
		name = r.Field
	case RuleZQL:
		name = r.Name
	default:
		return fmt.Sprintf("unknown type: %s", r.Kind)
	}
	return fmt.Sprintf("%s-%s", r.Kind, name)
}

func (r Rule) Marshal() ([]byte, error) {
	rec, err := resolver.NewMarshaler().MarshalRecord(r)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw := zngio.NewWriter(zio.NopCloser(&buf), zngio.WriterOpts{})
	if err := zw.Write(rec); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
