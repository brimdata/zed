package archive

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
	"github.com/brimsec/zq/zql"
)

type IndexQuery struct {
	indexName string
	patterns  []string
}

func ParseIndexQuery(indexName string, patterns []string) (IndexQuery, error) {
	if len(patterns) == 0 {
		return IndexQuery{}, zqe.E(zqe.Invalid, "no search patterns")
	}
	if indexName != "" {
		return IndexQuery{
			indexName: indexName,
			patterns:  patterns,
		}, nil
	}
	if len(patterns) != 1 {
		return IndexQuery{}, zqe.E(zqe.Invalid, "standard index supports exactly one search pattern")
	}
	in := patterns[0]

	v := strings.Split(in, "=")
	if len(v) != 2 {
		return IndexQuery{}, zqe.E(zqe.Invalid, "malformed standard index query")
	}
	fieldOrType := v[0]
	var path string
	if fieldOrType[0] == ':' {
		typ, err := resolver.NewContext().LookupByName(fieldOrType[1:])
		if err != nil {
			return IndexQuery{}, err
		}
		path = typeZdxName(typ)
	} else {
		path = fieldZdxName(fieldOrType)
	}
	return IndexQuery{
		indexName: path,
		patterns:  []string{v[1]},
	}, nil
}

func NewRule(pattern string) (*Rule, error) {
	if pattern[0] == ':' {
		return NewTypeRule(pattern[1:])
	}
	return NewFieldRule(pattern)
}

// we make the framesize here larger than the writer framesize
// since the writer always writes a bit past the threshold
const framesize = 32 * 1024 * 2

const keyName = "key"

var keyAst = ast.Assignment{
	Target: "key",
	Expr:   &ast.FieldRead{Field: "key"},
}

// NewFieldRule creates an indexing rule that will index all fields of
// the type passed in as argument.
func NewTypeRule(typeName string) (*Rule, error) {
	typ, err := resolver.NewContext().LookupByName(typeName)
	if err != nil {
		return nil, err
	}
	c := ast.SequentialProc{
		Procs: []ast.Proc{
			&typeSplitterNode{
				key:      keyName,
				typeName: typeName,
			},
			&ast.GroupByProc{
				Keys: []ast.Assignment{keyAst},
			},
			&ast.SortProc{},
		},
	}
	return newRuleAST(&c, typeZdxName(typ), []string{keyName}, framesize)
}

// NewFieldRule creates an indexing rule that will index the field passed in as argument.
// It is currently an error to try to index a field name that appears as different types.
func NewFieldRule(fieldName string) (*Rule, error) {
	c := ast.SequentialProc{
		Procs: []ast.Proc{
			&fieldCutterNode{
				field: fieldName,
				out:   keyName,
			},
			&ast.GroupByProc{
				Keys: []ast.Assignment{keyAst},
			},
			&ast.SortProc{},
		},
	}
	return newRuleAST(&c, fieldZdxName(fieldName), []string{keyName}, framesize)
}

// Rule contains the runtime configuration for an indexing rule.
type Rule struct {
	proc      ast.Proc
	path      string
	framesize int
	keys      []string
}

func newRuleAST(proc ast.Proc, path string, keys []string, framesize int) (*Rule, error) {
	if path == "" {
		return nil, fmt.Errorf("zql indexing rule requires an output path")
	}
	return &Rule{
		proc:      proc,
		path:      path,
		framesize: framesize,
		keys:      keys,
	}, nil
}

func NewZqlRule(s, path string, keys []string, framesize int) (*Rule, error) {
	proc, err := zql.ParseProc(s)
	if err != nil {
		return nil, err
	}
	return newRuleAST(proc, path, keys, framesize)
}

func (f *Rule) Path(dir iosrc.URI) iosrc.URI {
	return dir.AppendPath(f.path)
}
