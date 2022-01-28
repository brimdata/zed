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
	s := rec.Deref("foo").AsString()
	assert.Equal(t, s, "hello")
	b := rec.Deref("bar").AsBool()
	assert.Equal(t, b, true)
}

func TestNonRecordDeref(t *testing.T) {
	const input = `
1
192.168.1.1
null
[1,2,3]
|[1,2,3]|`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	for {
		val, err := reader.Read()
		if val == nil {
			break
		}
		require.NoError(t, err)
		v := val.Deref("foo")
		require.Nil(t, v)
	}
}
