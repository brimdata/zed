package zx_test

import (
	"testing"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zx"
	"github.com/stretchr/testify/require"
)

func TestCoerceDuration(t *testing.T) {
	var interval int64 = 60_000_000_000
	runcase(t, "Uint64", zng.NewUint64(60), interval)
	runcase(t, "Float64", zng.NewFloat64(60), interval)
	runcase(t, "Duration", zng.NewDuration(60*1e9), interval)

	// can't coerce
	notcase(t, "NotPort", zng.NewPort(60))
}

func runcase(t *testing.T, name string, in zng.Value, expected int64) {
	t.Run(name, func(t *testing.T) {
		val, ok := zx.CoerceToDuration(in)
		require.True(t, ok, "coercion succeeded")
		require.Equal(t, expected, val, "coerced value is correct")
	})
}

func notcase(t *testing.T, name string, in zng.Value) {
	t.Run(name, func(t *testing.T) {
		_, ok := zx.CoerceToDuration(in)
		require.False(t, ok, "coercion should have failed")
	})
}
