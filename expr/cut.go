package expr

import (
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type cutFunc struct {
	*Cutter
}

func compileCut(zctx *resolver.Context, node ast.FunctionCall) (Evaluator, error) {
	var lhs []field.Static
	var rhs []Evaluator
	for _, expr := range node.Args {
		// This is a bit of a hack and could be cleaed up by re-factoring
		// CompileAssigment, but for now, we create an assigment expression
		// where the LHS and RHS are the same, so that cut(id.orig_h,_path)
		// gives a value of type record[id:record[orig_h:ip],_path:string]
		// with field names that are the same as the cut names.
		assignment := &ast.Assignment{LHS: expr, RHS: expr}
		compiled, err := CompileAssignment(zctx, assignment)
		if err != nil {
			return nil, err
		}
		lhs = append(lhs, compiled.LHS)
		rhs = append(rhs, compiled.RHS)
	}
	return &cutFunc{NewCutter(zctx, false, lhs, rhs)}, nil
}

func (c *cutFunc) Eval(rec *zng.Record) (zng.Value, error) {
	out, err := c.Cut(rec)
	if err != nil {
		return zng.Value{}, err
	}
	if out == nil {
		return zng.Value{}, ErrNoSuchField
	}
	return zng.Value{out.Type, out.Raw}, nil
}
