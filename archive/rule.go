package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

func ParsePattern(in string) (string, string, error) {
	v := strings.Split(in, "=")
	if len(v) != 2 {
		return "", "", errors.New("not a standard index search")
	}
	fieldOrType := v[0]
	var path string
	if fieldOrType[0] == ':' {
		typ, err := resolver.NewContext().LookupByName(fieldOrType[1:])
		if err != nil {
			return "", "", err
		}
		path = typeZdxName(typ)
	} else {
		path = fieldZdxName(fieldOrType)
	}
	return v[1], path, nil

}

// Rule is an interface for creating pattern-specific indexers and finders
// dynamically as directories are encountered.
type Rule interface {
	NewIndexer(zardir string) (zbuf.WriteCloser, error)
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

func (t *TypeRule) NewIndexer(dir string) (zbuf.WriteCloser, error) {
	zdxPath := filepath.Join(dir, typeZdxName(t.Type))
	// XXX DANGER, remove without warning, should we have a force flag?
	if err := zdx.Remove(zdxPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return NewTypeIndexer(zdxPath, t.Type), nil
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

func (f *FieldRule) NewIndexer(dir string) (zbuf.WriteCloser, error) {
	zdxPath := filepath.Join(dir, fieldZdxName(f.field))
	// XXX DANGER, remove without warning, should we have a force flag?
	if err := zdx.Remove(zdxPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return NewFieldIndexer(zdxPath, f.field, f.accessor), nil
}
