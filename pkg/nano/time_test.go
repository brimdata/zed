package nano_test

import (
	"testing"
	"time"

	"github.com/brimdata/zed/pkg/nano"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	successCases := []struct {
		input      string
		expectedTs nano.Ts
	}{
		{"0", 0},
		{"1425565514.419939", 1425565514419939000},
		{"001425565514.419939", 1425565514419939000},
		{"-1425565514.419939", -1425565514419939000},
		{"1e9", 1e9 * 1e9},
		{"1.123e8", 1.123e8 * 1e9},
		{"1.123e-5", nano.Ts(1.123e-5 * 1e9)},
	}
	for _, c := range successCases {
		ts, err := nano.Parse([]byte(c.input))
		assert.NoError(t, err, "input: %q", c.input)
		assert.Exactly(t, c.expectedTs, ts, "input: %q", c.input)
	}

	for _, input := range []string{"", " ", "a"} {
		_, err := nano.Parse([]byte(input))
		assert.Error(t, err, "input: %q", input)
	}
}

func TestParseMillis(t *testing.T) {
	successCases := []struct {
		input      string
		expectedTs nano.Ts
	}{
		{"0", 0},
		{"00", 0},
		{"2", 2 * 1_000_000},
		{"03", 3 * 1_000_000},
		{"1234567890", 1234567890 * 1_000_000},
	}
	for _, c := range successCases {
		ts, err := nano.ParseMillis([]byte(c.input))
		assert.NoError(t, err, "input: %q", c.input)
		assert.Exactly(t, c.expectedTs, ts, "input: %q", c.input)
	}

	for _, input := range []string{"", " ", "+1", "-1", "a", "1.2", "1579438676648060000"} {
		_, err := nano.ParseMillis([]byte(input))
		assert.Error(t, err, "input: %q", input)
	}
}

func TestStringFloat(t *testing.T) {
	ts := nano.Ts((time.Minute + 1) * -1)
	assert.Equal(t, "-60.000000001", ts.StringFloat())
	ts = nano.Ts((time.Minute + 10) * -1)
	assert.Equal(t, "-60.00000001", ts.StringFloat())
	ts = nano.Ts((time.Minute) * -1)
	assert.Equal(t, "-60", ts.StringFloat())
	ts = nano.Ts((time.Millisecond * 100) * -1)
	assert.Equal(t, "-0.1", ts.StringFloat())
}

func TestAppendFloat(t *testing.T) {
	ts := nano.Ts((time.Minute) * -1)
	assert.Equal(t, "-60.000000", string(ts.AppendFloat(nil, 6)))
	ts = nano.Ts((time.Minute + time.Millisecond))
	assert.Equal(t, "60.001000000", string(ts.AppendFloat(nil, 9)))
}
