package units

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytesAbrrev(t *testing.T) {
	require.Exactly(t, "2B", Bytes(2).Abbrev())
	require.Exactly(t, "1234B", Bytes(1234).Abbrev())
	require.Exactly(t, "12.34KB", Bytes(12340).Abbrev())
	require.Exactly(t, "1.23MB", Bytes(1_234_000).Abbrev())
	require.Exactly(t, "1.23GB", Bytes(1_234_000_000).Abbrev())
}
