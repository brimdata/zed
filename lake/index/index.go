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

type IndexKind string

const (
	IndexType  IndexKind = "type"
	IndexField IndexKind = "field"
	IndexZed   IndexKind = "zed"
)

// Index contains the runtime configuration for an index.
type Index struct {
	Framesize int            `zng:"framesize,omitempty"`
	ID        ksuid.KSUID    `zng:"id"`
	Name      string         `zng:"name"`
	Keys      []field.Static `zng:"keys,omitempty"`
	Kind      IndexKind      `zng:"kind"`
	Value     string         `zng:"type"`
}

func ParseIndex(pattern string) (Index, error) {
	if pattern[0] == ':' {
		typ, err := zson.NewContext().LookupByName(pattern[1:])
		if err != nil {
			return Index{}, err
		}
		return NewTypeIndex(typ), nil
	}
	return NewFieldIndex(pattern), nil
}

func NewTypeIndex(typ zng.Type) Index {
	return Index{
		ID:    ksuid.New(),
		Kind:  IndexType,
		Value: tzngio.TypeString(typ),
	}
}

// NewFieldIndex creates an index that will index the field passed in as argument.
// It is currently an error to try to index a field name that appears as different types.
func NewFieldIndex(fieldName string) Index {
	return Index{
		ID:    ksuid.New(),
		Kind:  IndexField,
		Value: fieldName,
	}
}

func UnmarshalIndex(b []byte) (Index, error) {
	zctx := zson.NewContext()
	zr := zngio.NewReader(bytes.NewReader(b), zctx)
	rec, err := zr.Read()
	if err != nil {
		return Index{}, err
	}
	r := Index{}
	return r, resolver.UnmarshalRecord(rec, &r)
}

func NewZedIndex(prog, name string, keys []field.Static) (Index, error) {
	// make sure it compiles
	if _, err := compiler.ParseProc(prog); err != nil {
		return Index{}, err
	}
	return Index{
		ID:    ksuid.New(),
		Keys:  keys,
		Kind:  IndexZed,
		Name:  name,
		Value: prog,
	}, nil
}

// Equivalent determine if the provided Index is equivalent to the receiver. It
// should used to check if a Definition already contains and equivalent index.
func (i Index) Equivalent(r2 Index) bool {
	if i.Kind != r2.Kind || i.Value != r2.Value {
		return false
	}
	if i.Kind == IndexZed {
		return i.Name == r2.Name
	}
	return true
}

func (i Index) Proc() (ast.Proc, error) {
	switch i.Kind {
	case IndexType:
		return i.typeProc()
	case IndexField:
		return i.fieldProc()
	case IndexZed:
		return i.zqlProc()
	default:
		return nil, fmt.Errorf("unknown index kind: %s", i.Kind)
	}
}

var keyName = field.New("key")

var keyAst = ast.Assignment{
	LHS: ast.NewDotExpr(keyName),
	RHS: ast.NewDotExpr(keyName),
}
var countAst = ast.NewAggAssignment("count", nil, nil)

// NewFieldRule creates an index that will index all fields of
// the type passed in as argument.
func (i Index) typeProc() (ast.Proc, error) {
	return &ast.Sequential{
		Kind: "Sequential",
		Procs: []ast.Proc{
			&ast.TypeSplitter{
				Key:      keyName,
				TypeName: i.Value,
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

func (i Index) fieldProc() (ast.Proc, error) {
	return &ast.Sequential{
		Kind: "Sequential",
		Procs: []ast.Proc{
			&ast.FieldCutter{
				Field: field.Dotted(i.Value),
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

func (i Index) zqlProc() (ast.Proc, error) {
	return compiler.ParseProc(i.Value)
}

func (i Index) String() string {
	name := i.Value
	if i.Kind == IndexZed {
		name = i.Name
	}
	return fmt.Sprintf("%s->%s", i.Kind, name)
}

type Indices []Index

func (indices Indices) Lookup(id ksuid.KSUID) *Index {
	if i := indices.indexOf(id); i >= 0 {
		return &indices[i]
	}
	return nil
}

// Add checks if Indices already has an equivalent Index and if it does not
// returns Indices with the Index appended to it. Returns a non-nil Index pointer
// if an equivalent Index is found.
func (indices Indices) Add(index Index) (Indices, *Index) {
	for _, i := range indices {
		if i.Equivalent(index) {
			return indices, &i
		}
	}
	return append(indices, index), nil
}

// LookupDelete checks the Indices list for a index matching the provided ID
// returning the deleted index if found.
func (indices Indices) LookupDelete(id ksuid.KSUID) (Indices, *Index) {
	if i := indices.indexOf(id); i >= 0 {
		index := indices[i]
		return append(indices[:i], indices[i+1:]...), &index
	}
	return indices, nil
}

func (indices Indices) indexOf(id ksuid.KSUID) int {
	for i, index := range indices {
		if index.ID == id {
			return i
		}
	}
	return -1
}

func (indices Indices) IDs() []ksuid.KSUID {
	ids := make([]ksuid.KSUID, len(indices))
	for k, index := range indices {
		ids[k] = index.ID
	}
	return ids
}
