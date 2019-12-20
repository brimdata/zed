package nano_test

import (
	"testing"
	"time"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/stretchr/testify/assert"
)

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

	for _, input := range []string{"", " ", "+1", "-1", "a", "1.2"} {
		_, err := nano.ParseMillis([]byte(input))
		assert.Error(t, err, "input: %q", input)
	}
}

func TestStringFloat(t *testing.T) {
	ts := nano.Ts((time.Minute + 1) * -1)
	assert.Equal(t, "-60.000000001", ts.StringFloat())
	ts = nano.Ts((time.Minute + 10) * -1)
	assert.Equal(t, "-60.00000001", ts.StringFloat())
}
