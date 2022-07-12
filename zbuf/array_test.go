package zbuf

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/require"
)

func TestArrayWriteCopiesValueBytes(t *testing.T) {
	var a Array
	val := zed.NewString("old")
	a.Write(val)
	copy(val.Bytes, zed.EncodeString("new"))
	require.Equal(t, zed.NewString("old"), &a.Values()[0])
}
