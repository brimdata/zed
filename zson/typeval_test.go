package zson_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/require"
)

func TestTypeValue(t *testing.T) {
	const s = "{A:{B:int64},C:int32}"
	zctx := zed.NewContext()
	typ, err := zson.ParseType(zctx, s)
	require.NoError(t, err)
	arena := zed.NewArena()
	defer arena.Unref()
	tv := zctx.LookupTypeValue(arena, typ)
	require.Exactly(t, s, zson.FormatTypeValue(tv.Bytes()))
}

func TestTypeValueCrossContext(t *testing.T) {
	const s = "{A:{B:int64},C:int32}"
	typ, err := zson.ParseType(zed.NewContext(), s)
	require.NoError(t, err)
	arena := zed.NewArena()
	defer arena.Unref()
	tv := zed.NewContext().LookupTypeValue(arena, typ)
	require.Exactly(t, s, zson.FormatTypeValue(tv.Bytes()))
}
