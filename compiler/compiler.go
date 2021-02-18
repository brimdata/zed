package compiler

import (
	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/compiler/kernel"
	"github.com/brimsec/zq/compiler/semantic"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
)

// ParseProc() is an entry point for use from external go code,
// mostly just a wrapper around Parse() that casts the return value.
//func ParseProc(query string, opts ...zql.Option) (ast.Proc, error) {
func ParseProc(query string) (ast.Proc, error) {
	parsed, err := zql.Parse("", []byte(query), zql.Entrypoint("start"))
	if err != nil {
		return nil, err
	}
	return ast.UnpackProc(nil, parsed)
}

func ParseExpression(expr string) (ast.Expression, error) {
	parsed, err := zql.Parse("", []byte(expr), zql.Entrypoint("Expr"))
	if err != nil {
		return nil, err
	}
	return ast.UnpackExpression(nil, parsed)
}

func ParseProgram(z string) (*ast.Program, error) {
	parsed, err := zql.Parse("", []byte(z), zql.Entrypoint("Program"))
	if err != nil {
		proc, nerr := ParseProc(z)
		if nerr != nil {
			return nil, err
		}
		return &ast.Program{Entry: proc}, nil
	}
	return ast.UnpackProgram(nil, parsed)
}

func ParseToObject(expr, entry string) (interface{}, error) {
	return zql.Parse("", []byte(expr), zql.Entrypoint(entry))
}

// MustParseProc is functionally the same as ParseProc but panics if an error
// is encountered.
func MustParseProc(query string) ast.Proc {
	proc, err := ParseProc(query)
	if err != nil {
		panic(err)
	}
	return proc
}

// XXX These functions will all get reworked in a subsequent PR when
// the semantic pass converts an AST to a flow DSL.

func Optimize(zctx *resolver.Context, program ast.Proc, sortKey field.Static, sortReversed bool) (*kernel.Filter, ast.Proc) {
	return semantic.Optimize(zctx, program, sortKey, sortReversed)
}

func IsParallelizable(p ast.Proc, inputSortField field.Static, inputSortReversed bool) bool {
	return semantic.IsParallelizable(p, inputSortField, inputSortReversed)
}

func Parallelize(p ast.Proc, N int, inputSortField field.Static, inputSortReversed bool) (*ast.SequentialProc, bool) {
	return semantic.Parallelize(p, N, inputSortField, inputSortReversed)
}

func NewFilter(zctx *resolver.Context, ast ast.Expression) *kernel.Filter {
	return kernel.NewFilter(zctx, ast)
}
func Compile(custom kernel.Hook, node ast.Proc, pctx *proc.Context, parents []proc.Interface) ([]proc.Interface, error) {
	return kernel.Compile(custom, node, pctx, nil, parents)
}

func CompileAssignments(dsts []field.Static, srcs []field.Static) ([]field.Static, []expr.Evaluator) {
	return kernel.CompileAssignments(dsts, srcs)
}

func MustParseProgram(query string) *ast.Program {
	p, err := ParseProgram(query)
	if err != nil {
		if proc, err := ParseProc(query); err == nil {
			return &ast.Program{Entry: proc}
		}
		panic(err)
	}
	return p
}
