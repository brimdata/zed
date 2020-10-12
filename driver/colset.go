package driver

import (
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/field"
)

type Colset struct {
	table map[string]struct{}
	key   []byte
}

func newColset() *Colset {
	return &Colset{table: make(map[string]struct{})}
}

func (c *Colset) keyOf(f field.Static) []byte {
	key := c.key[:0]
	for k, s := range f {
		key = append(key, []byte(s)...)
		if k > 0 {
			key = append(key, 0)
		}
		if len(key) > 10000 {
			panic("key too long")
		}
	}
	c.key = key
	return key
}

// Add adds the field access determined by the expression to the table if
// it can be statically determined to be a field reference, in which case
// true is returned.  Otherwise, false is returned to indicate that it is
// unknown what the field name expressiion might be at run time.
func (c *Colset) Add(name ast.Expression) bool {
	f, ok := ast.DotExprToField(name)
	if !ok {
		return false
	}
	c.table[string(c.keyOf(f))] = struct{}{}
	return true
}

func (c *Colset) Equal(to *Colset) bool {
	if c == nil {
		return to == nil
	}
	if len(c.table) != len(to.table) {
		return false
	}
	for k := range c.table {
		if _, ok := to.table[k]; !ok {
			return false
		}
	}
	return true
}
