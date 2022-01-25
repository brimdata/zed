package zed_test

import (
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordAccessNamed(t *testing.T) {
	const input = `{foo:"hello" (=zfile),bar:true (=zbool)} (=0)`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	rec, err := reader.Read()
	require.NoError(t, err)
	s, err := rec.AccessString("foo")
	require.NoError(t, err)
	assert.Equal(t, s, "hello")
	b, err := rec.AccessBool("bar")
	require.NoError(t, err)
	assert.Equal(t, b, true)
}
