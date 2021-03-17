package nano_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var boomerangs = []string{
	"1ns",
	"1us",
	"1ms",
	"1s",
	"1m",
	"1h",
	"2y3d5h24m1s",
	"100us",
	"123ns",
	"123ms",
	"1.3s",
	"0s",
	"31y259d1h46m40s",
	"2h43m9.993714061s",
}

func TestDuration(t *testing.T) {
	d, err := nano.ParseDuration("1ms")
	require.NoError(t, err)
	assert.Exactly(t, nano.Duration(time.Millisecond), d)
	for _, s := range boomerangs {
		checkdur(t, s, s)
	}
	checkdur(t, "1230ms", "1.23s")
	checkdur(t, "0ns", "0s")
	checkdur(t, "1s300ms", "1.3s")
	checkdur(t, "2716us", "2.716ms")
	checkdur(t, "1230us", "1.23ms")
	checkdur(t, "11230us", "11.23ms")
	checkdur(t, "111230us", "111.23ms")
	checkdur(t, "1234ns", "1.234us")
	checkdur(t, "", "0s")
}

func checkdur(t *testing.T, in, expected string) {
	d, err := nano.ParseDuration(in)
	require.NoError(t, err)
	actual := d.String()
	assert.Exactly(t, expected, actual)
}

func TestMarshalDuration(t *testing.T) {
	for _, s := range boomerangs {
		d, err := nano.ParseDuration(s)
		require.NoError(t, err)
		b, err := json.Marshal(&d)
		require.NoError(t, err)
		var actual nano.Duration
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		assert.Exactly(t, d, actual)
	}
}
