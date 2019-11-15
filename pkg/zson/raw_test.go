package zson

import (
	"testing"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRawAndTsFromJSON(t *testing.T) {
	typ, err := zeek.LookupType("record[b:bool,i:int,s:set[bool],ts:time,v:vector[int]]")
	require.NoError(t, err)
	d := NewDescriptor(typ.(*zeek.TypeRecord))
	tsCol, ok := d.ColumnOfField("ts")
	require.True(t, ok)

	const expectedTs = nano.Ts(1573860644637486000)
	cases := []struct {
		input      string
		expectedTs nano.Ts
	}{
		{`{"ts":"2019-11-15T23:30:44.637486Z"}`, expectedTs},  // JSON::TS_ISO8601
		{`{"ts":1573860644.637486}`, expectedTs},              // JSON::TS_EPOCH
		{`{"ts":1573860644637}`, expectedTs.Trunc(1_000_000)}, // JSON::TS_MILLIS
	}
	for _, c := range cases {
		raw, ts, _, err := NewRawAndTsFromJSON(d, tsCol, []byte(c.input))
		assert.NoError(t, err, "input: %s", c.input)
		assert.Exactly(t, c.expectedTs, ts, "input: %s", c.input)
		actualZeekValue := NewRecord(d, ts, raw).ValueByColumn(tsCol)
		require.NotNil(t, actualZeekValue, "input: %s", c.input)
		assert.Exactly(t, c.expectedTs, actualZeekValue.(*zeek.Time).Native, "input: %s", c.input)
	}
}
