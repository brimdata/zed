package csvio

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocess(t *testing.T) {
	const input = `
field1,"field"2,field"3" my friend
field4,"field"5 with "multiple" quotes "to" escape,field6`
	const expected = `
field1,field2,field3 my friend
field4,field5 with multiple quotes to escape,field6`
	p := newPreprocess(strings.NewReader(input))
	var buf bytes.Buffer
	_, err := io.Copy(&buf, p)
	require.NoError(t, err)
	assert.Equal(t, expected, buf.String())
}
