package zed_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/require"
)

func TestTypeValue(t *testing.T) {
	const s = "{A:{B:int64},C:int32}"
	zctx := zed.NewContext()
	typ, err := zson.ParseType(zed.NewContext(), s)
	require.NoError(t, err)
	tv := zctx.LookupTypeValue(typ)
	require.Exactly(t, s, zed.FormatTypeValue(tv.Bytes))
}

func TestTypeValueCrossContext(t *testing.T) {
	const s = "{A:{B:int64},C:int32}"
	typ, err := zson.ParseType(zed.NewContext(), s)
	require.NoError(t, err)
	tv := zed.NewContext().LookupTypeValue(typ)
	require.Exactly(t, s, zed.FormatTypeValue(tv.Bytes))
}
