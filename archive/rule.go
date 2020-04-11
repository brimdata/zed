package archive

import (
	"fmt"
	"path/filepath"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Rule is a interface for creating pattern-specific indexers and finders
// dynamically as directories are encountered.
type Rule interface {
	NewIndexer(dir string) Indexer
	NewFinder(dir string) *zdx.Finder
}

func NewRule(pattern string) (Rule, error) {
	if pattern[0] == ':' {
		return NewTypeRule(pattern[1:])
	}
	return NewFieldRule(pattern)
}

// TypeRule provides a means to generate Indexers and Finders for a type-specific
// rule. Each TypeRule instance is configured with a field name and the "new" methods
// create Indexers and Finders that operate according to this type.
type TypeRule struct {
	Type zng.Type
}

func NewTypeRule(typeName string) (*TypeRule, error) {
	typ, err := resolver.NewContext().LookupByName(typeName)
	if err != nil {
		return nil, err
	}
	return &TypeRule{
		Type: typ,
	}, nil
}

func (t *TypeRule) NewIndexer(dir string) Indexer {
	zdxPath := filepath.Join(dir, typeZdxName(t.Type))
	// XXX DANGER, remove without warning, should we have a force flag?
	zdx.Remove(zdxPath)
	return NewTypeIndexer(zdxPath, t.Type)
}

func (t *TypeRule) NewFinder(dir string) *zdx.Finder {
	zdxPath := filepath.Join(dir, typeZdxName(t.Type))
	return zdx.NewFinder(zdxPath)
}

// FieldRule provides a means to generate Indexers and Finders for a field-specific
// rules.  Each FieldRule is configured with a field name and the new methods
// create Indexers and Finders that operate on this field.
type FieldRule struct {
	field    string
	accessor expr.FieldExprResolver
}

func NewFieldRule(field string) (*FieldRule, error) {
	accessor := expr.CompileFieldAccess(field)
	if accessor == nil {
		return nil, fmt.Errorf("bad field syntax: %s", field)
	}
	return &FieldRule{
		field:    field,
		accessor: accessor,
	}, nil
}

func (f *FieldRule) NewIndexer(dir string) Indexer {
	zdxPath := filepath.Join(dir, fieldZdxName(f.field))
	// XXX DANGER, remove without warning, should we have a force flag?
	zdx.Remove(zdxPath)
	return NewFieldIndexer(zdxPath, f.field, f.accessor)
}

func (f *FieldRule) NewFinder(dir string) *zdx.Finder {
	zdxPath := filepath.Join(dir, fieldZdxName(f.field))
	return zdx.NewFinder(zdxPath)
}
