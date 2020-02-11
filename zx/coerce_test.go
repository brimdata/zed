package zx_test

import (
	"testing"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
	"github.com/stretchr/testify/require"
)

func TestCoerceDuration(t *testing.T) {
	interval := zng.NewDuration(60 * 1e9)
	runcase(t, "Int64", zng.NewInt(60), interval)
	runcase(t, "Uint64", zng.NewCount(60), interval)
	runcase(t, "Float64", zng.NewFloat64(60), interval)
	runcase(t, "Duration", zng.NewDuration(60*1e9), interval)

	// can't coerce
	notcase(t, "NotPort", zng.NewPort(60), zng.TypeDuration)
}

func runcase(t *testing.T, name string, in zng.Value, expected zng.Value) {
	t.Run(name, func(t *testing.T) {
		val, ok := zx.Coerce(in, expected.Type)
		require.Equal(t, true, ok)
		require.Equal(t, expected, val)
	})
}

func notcase(t *testing.T, name string, in zng.Value, typ zng.Type) {
	t.Run(name, func(t *testing.T) {
		_, ok := zx.Coerce(in, typ)
		require.Equal(t, false, ok)
	})
}
