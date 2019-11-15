package nano

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMillis(t *testing.T) {
	successCases := []struct {
		input      string
		expectedTs Ts
	}{
		{"0", 0},
		{"00", 0},
		{"2", 2 * 1_000_000},
		{"03", 3 * 1_000_000},
		{"1234567890", 1234567890 * 1_000_000},
	}
	for _, c := range successCases {
		ts, err := ParseMillis([]byte(c.input))
		assert.NoError(t, err, "input: %q", c.input)
		assert.Exactly(t, c.expectedTs, ts, "input: %q", c.input)
	}

	for _, input := range []string{"", " ", "+1", "-1", "a", "1.2"} {
		_, err := ParseMillis([]byte(input))
		assert.Error(t, err, "input: %q", input)
	}
}
