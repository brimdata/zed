package zed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextLookupTypeNamedAndLookupTypeDef(t *testing.T) {
	zctx := NewContext()

	assert.Nil(t, zctx.LookupTypeDef("x"))

	named1 := zctx.LookupTypeNamed("x", TypeNull)
	assert.Same(t, named1, zctx.LookupTypeDef("x"))

	named2 := zctx.LookupTypeNamed("x", TypeInt8)
	assert.Same(t, named2, zctx.LookupTypeDef("x"))

	named3 := zctx.LookupTypeNamed("x", TypeNull)
	assert.Same(t, named3, zctx.LookupTypeDef("x"))
	assert.Same(t, named3, named1)
}
