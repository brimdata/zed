package expr

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetContext(t *testing.T) {
	var ectx ResetContext
	var val *zed.Value

	// Test empty value with ResetContext.buf == nil.
	require.Nil(t, ectx.buf)
	val = zed.NewBytes([]byte{})
	assert.Equal(t, val, ectx.NewValue(val.Type, val.Bytes()))
	assert.Equal(t, val, ectx.CopyValue(*val))

	val = zed.NewBytes([]byte{'b'})
	assert.Equal(t, val, ectx.NewValue(val.Type, val.Bytes()))
	assert.Equal(t, val, ectx.CopyValue(*val))

	// Test null value with ResetContext.buf != nil.
	require.NotNil(t, ectx.buf)
	val = zed.NewBytes(nil)
	assert.Equal(t, val, ectx.NewValue(val.Type, val.Bytes()))
	assert.Equal(t, val, ectx.CopyValue(*val))
}
