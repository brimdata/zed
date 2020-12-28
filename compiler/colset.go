package compiler

import (
	"reflect"

	"github.com/brimsec/zq/ast"
)

type Colset map[string]struct{}

func newColset() Colset {
	return make(map[string]struct{})
}

// Add adds the field access determined by the expression to the table if
// it can be statically determined to be a field reference, in which case
// true is returned.  Otherwise, false is returned to indicate that it is
// unknown what the field name expressiion might be at run time.
func (c Colset) Add(name ast.Expression) bool {
	f, ok := ast.DotExprToField(name)
	if !ok {
		return false
	}
	c[f.String()] = struct{}{}
	return true
}

func (c Colset) Equal(to Colset) bool {
	return reflect.DeepEqual(c, to)
}
