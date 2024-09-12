package zbuf

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/require"
)

func TestArrayWriteCopiesValueBytes(t *testing.T) {
	var a Array
	val := zed.NewBytes([]byte{0})
	a.Write(val)
	copy(val.Bytes(), zed.EncodeBytes([]byte{1}))
	require.Equal(t, zed.NewBytes([]byte{0}), a.Values()[0])
}
