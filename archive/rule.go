package archive

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/field"
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
		path = typeMicroIndexName(typ)
	} else {
		path = fieldMicroIndexName(fieldOrType)
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

var keyName = field.New("key")

var keyAst = ast.Assignment{
	LHS: ast.NewDotExpr(field.New("key")),
	RHS: ast.NewDotExpr(field.New("key")),
}
var countAst = ast.NewReducerAssignment("count", nil, nil)

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
				Keys:     []ast.Assignment{keyAst},
				Reducers: []ast.Assignment{countAst},
			},
			&ast.SortProc{Fields: []ast.Expression{ast.NewDotExpr(field.New("key"))}},
		},
	}
	return NewRuleAST("type", &c, typeMicroIndexName(typ), []field.Static{keyName}, framesize)
}

// NewFieldRule creates an indexing rule that will index the field passed in as argument.
// It is currently an error to try to index a field name that appears as different types.
func NewFieldRule(fieldName string) (*Rule, error) {
	c := ast.SequentialProc{
		Procs: []ast.Proc{
			&fieldCutterNode{
				field: field.Dotted(fieldName),
				out:   keyName,
			},
			&ast.GroupByProc{
				Keys:     []ast.Assignment{keyAst},
				Reducers: []ast.Assignment{countAst},
			},
			&ast.SortProc{Fields: []ast.Expression{ast.NewDotExpr(field.New("key"))}},
		},
	}
	return NewRuleAST("field", &c, fieldMicroIndexName(fieldName), []field.Static{keyName}, framesize)
}

// Rule contains the runtime configuration for an indexing rule.
type Rule struct {
	typ       string
	proc      ast.Proc
	path      string
	framesize int
	keys      []field.Static
}

func NewRuleAST(typ string, proc ast.Proc, path string, keys []field.Static, fsize int) (*Rule, error) {
	if path == "" {
		return nil, fmt.Errorf("zql indexing rule requires an output path")
	}
	if fsize == 0 {
		fsize = framesize
	}
	return &Rule{
		typ:       typ,
		proc:      proc,
		path:      path,
		framesize: fsize,
		keys:      keys,
	}, nil
}

func NewZqlRule(s, path string, keys []field.Static, framesize int) (*Rule, error) {
	proc, err := zql.ParseProc(s)
	if err != nil {
		return nil, err
	}
	return NewRuleAST("zql", proc, path, keys, framesize)
}

func (f *Rule) Path(dir iosrc.URI) iosrc.URI {
	return dir.AppendPath(f.path)
}
