package queryio_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkRecord(t *testing.T, s string) *zed.Value {
	r := zsonio.NewReader(zed.NewContext(), strings.NewReader(s))
	rec, err := r.Read()
	require.NoError(t, err)
	return rec
}

func TestZJSONWriter(t *testing.T) {
	const record = `{x:1}`
	const expected = `
{"type":"QueryChannelSet","value":{"channel_id":1}}
{"type":{"kind":"record","id":30,"fields":[{"name":"x","type":{"kind":"primitive","name":"int64"}}]},"value":["1"]}
{"type":"QueryChannelEnd","value":{"channel_id":1}}
{"type":"QueryError","value":{"error":"test.err"}}
`
	var buf bytes.Buffer
	w := queryio.NewZJSONWriter(&buf)
	err := w.WriteControl(api.QueryChannelSet{ChannelID: 1})
	require.NoError(t, err)
	err = w.Write(mkRecord(t, record))
	require.NoError(t, err)
	err = w.WriteControl(api.QueryChannelEnd{ChannelID: 1})
	require.NoError(t, err)
	err = w.WriteControl(api.QueryError{Error: "test.err"})
	require.NoError(t, err)
	assert.Equal(t, expected, "\n"+buf.String())
}
