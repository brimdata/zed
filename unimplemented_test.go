package super_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superdb/super"
	"github.com/superdb/super/sup"
)

func TestNewUnimplemented(t *testing.T) {
	sctx := super.NewContext()
	val := sctx.NewUnimplemented(super.TypeDate, super.EncodeInt(20000))
	require.True(t, val.IsError())
	require.Equal(t, `error({unimplemented:<date>,bytes:0x409c})`, sup.FormatValue(val))
	val = sctx.NewUnimplemented(sctx.LookupTypeDecimal(10, 2), super.EncodeInt(1234))
	require.Equal(t, `error({unimplemented:<decimal(10,2)>,bytes:0xa409})`, sup.FormatValue(val))
}

func TestReservedTypeIDs(t *testing.T) {
	for _, name := range []string{"uint128", "int128", "timestamp(0)", "timestamp(3)", "timestamp(6)", "timestamp", "timestamp_ntz", "date", "time", "interval", "uuid", "geometry", "geography"} {
		typ := super.LookupPrimitive(name)
		require.NotNil(t, typ, name)
		require.Equal(t, name, super.PrimitiveName(typ))
		byID, err := super.LookupPrimitiveByID(typ.ID())
		require.NoError(t, err)
		require.Equal(t, typ, byID)
		require.Less(t, typ.ID(), super.IDTypeComplex)
	}
	require.True(t, super.IsUnimplemented(super.TypeDate))
	require.False(t, super.IsUnimplemented(super.TypeTimestamp))
}
