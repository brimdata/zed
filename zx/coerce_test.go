package zx_test

import (
	"testing"

	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zx"
	"github.com/stretchr/testify/require"
)

func TestCoerceDuration(t *testing.T) {
	t.Run("Int", func(t *testing.T) {
		val := int64(60)
		expected := zng.NewInterval(val * 1e9)
		interval, ok := zx.Coerce(zng.NewInt(val), zng.TypeInterval)
		require.Equal(t, true, ok)
		require.Equal(t, expected, interval)
	})
	t.Run("Uint", func(t *testing.T) {
		val := uint64(60)
		expected := zng.NewInterval(int64(val * 1e9))
		interval, ok := zx.Coerce(zng.NewCount(val), zng.TypeInterval)
		require.Equal(t, true, ok)
		require.Equal(t, expected, interval)
	})
}
