package zson_test

import (
	"testing"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zson"
	"github.com/stretchr/testify/require"
)

func TestTable(t *testing.T) {
	zctx := resolver.NewContext()
	table := zson.NewTypeTable(zctx)
	typ, err := table.LookupType("({path:string,x:int32})")
	require.NoError(t, err)

	check := zctx.MustLookupTypeRecord([]zng.Column{
		zng.Column{"path", zng.TypeString},
		zng.Column{"x", zng.TypeInt32},
	})
	require.True(t, zng.SameType(typ, check))
}
