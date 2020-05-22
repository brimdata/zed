package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
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
	Path(zardir string) string
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

func (t *TypeRule) Path(dir string) string {
	return filepath.Join(dir, typeZdxName(t.Type))
}

func (t *TypeRule) NewIndexer(dir string) (zbuf.WriteCloser, error) {
	zdxPath := t.Path(dir)
	// XXX DANGER, remove without warning, should we have a force flag?
	if err := zdx.Remove(zdxPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return NewTypeIndexer(zdxPath, t.Type), nil
}

// NewFieldRule creates an indexing rule that will index the field passed in as argument.
func NewFieldRule(field string) (*ZqlRule, error) {
	c := ast.SequentialProc{
		Procs: []ast.Proc{
			&ast.CutProc{
				Fields: []string{field},
			},
			&ast.GroupByProc{
				Keys: []string{field},
			},
			&ast.SortProc{},
		},
	}

	return newZqlRuleAST(&c, fieldZdxName(field), []string{field}, framesize)
}

// xxx comment
// ZqlRule provides a means to generate Indexers for a zql rule.
// rules.  Each ZqlRule is configured with a field name and the new methods
// create Indexers and Finders that operate on this field.
type ZqlRule struct {
	proc      ast.Proc
	path      string
	framesize int
	keys      []string
}

func newZqlRuleAST(proc ast.Proc, path string, keys []string, framesize int) (*ZqlRule, error) {
	if path == "" {
		return nil, fmt.Errorf("zql indexing rule requires an output path")
	}
	return &ZqlRule{
		proc:      proc,
		path:      path,
		framesize: framesize,
		keys:      keys,
	}, nil
}

func NewZqlRule(s, path string, keys []string, framesize int) (*ZqlRule, error) {
	proc, err := zql.ParseProc(s)
	if err != nil {
		return nil, err
	}
	return newZqlRuleAST(proc, path, keys, framesize)
}

func (f *ZqlRule) Path(dir string) string {
	return filepath.Join(dir, f.path)
}

func (f *ZqlRule) NewIndexer(dir string) (zbuf.WriteCloser, error) {
	// once this is all done, there probably won't be a NewIndexer on the interface anymore
	panic("zqlRule: no NewIndexer")
}
