package expr

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetContext(t *testing.T) {
	var ectx ResetContext
	val := zed.NewBytes(nil)

	// Test empty value with ResetContext.buf == nil.
	require.Nil(t, ectx.buf)
	val.Bytes = []byte{}
	assert.Equal(t, val, ectx.NewValue(val.Type, val.Bytes))
	assert.Equal(t, val, ectx.CopyValue(val))

	val.Bytes = []byte{'b'}
	assert.Equal(t, val, ectx.NewValue(val.Type, val.Bytes))
	assert.Equal(t, val, ectx.CopyValue(val))

	// Test null value with ResetContext.buf != nil.
	require.NotNil(t, ectx.buf)
	val.Bytes = nil
	assert.Equal(t, val, ectx.NewValue(val.Type, val.Bytes))
	assert.Equal(t, val, ectx.CopyValue(val))
}
