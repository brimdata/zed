package zeekio

import (
	"strings"
	"testing"
	"time"

	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReaderCRLF(t *testing.T) {
	input := `
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	a
#fields	ts	i
#types	time	int
10.000000	1
`
	input = strings.ReplaceAll(input, "\n", "\r\n")
	r, err := NewReader(strings.NewReader(input), resolver.NewContext())
	require.NoError(t, err)
	rec, err := r.Read()
	require.NoError(t, err)
	ts, err := rec.AccessTime("ts")
	require.NoError(t, err)
	assert.Exactly(t, 10*nano.Ts(time.Second), ts)
	d, err := rec.AccessInt("i")
	require.NoError(t, err)
	assert.Exactly(t, int64(1), d)
	rec, err = r.Read()
	require.NoError(t, err)
	assert.Nil(t, rec)
}
