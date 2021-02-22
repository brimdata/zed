package resolver

import (
	"errors"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zson"
)

var (
	ErrAliasExists = errors.New("alias exists with different type")
)

type TypeResolver interface {
	Lookup(string) (zng.Type, error)
}

// A Context manages the mapping between small-integer descriptor identifiers
// and zng descriptor objects, which hold the binding between an identifier
// and a zng.Type.
type Context struct {
	*zson.Context
}

func NewContext() *Context {
	return &Context{
		Context: zson.NewContext(),
	}
}

func (c *Context) NewTypeTable() *zson.TypeTable {
	return zson.NewTypeTable(c.Context)
}
