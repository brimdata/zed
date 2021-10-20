package queryio_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/pkg/test"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkRecord(t *testing.T, s string) *zed.Value {
	r := zson.NewReader(strings.NewReader(s), zed.NewContext())
	rec, err := r.Read()
	require.NoError(t, err)
	return rec
}

func TestZJSONWriter(t *testing.T) {
	const record = `{x:1}`
	const expected = `
{"kind":"QueryChannelSet","value":{"channel_id":1}}
{"kind":"Object","value":{"schema":"23","types":[{"kind":"typedef","name":"23","type":{"kind":"record","fields":[{"name":"x","type":{"kind":"primitive","name":"int64"}}]}}],"values":["1"]}}
{"kind":"QueryChannelEnd","value":{"channel_id":1}}
{"kind":"QueryError","value":{"error":"test.err"}}
`
	var buf bytes.Buffer
	w := queryio.NewZJSONWriter(&buf)
	err := w.WriteControl(api.QueryChannelSet{1})
	require.NoError(t, err)
	err = w.Write(mkRecord(t, record))
	require.NoError(t, err)
	err = w.WriteControl(api.QueryChannelEnd{1})
	require.NoError(t, err)
	err = w.WriteControl(api.QueryError{"test.err"})
	require.NoError(t, err)
	assert.Equal(t, test.Trim(expected), buf.String())
}
