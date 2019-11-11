package zql

import (
	"bufio"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	file, err := os.Open("valid.zql")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		_, err := Parse("", line)
		assert.NoError(t, err, "zql: %q", line)
	}
}

func TestInvalid(t *testing.T) {
	file, err := os.Open("invalid.zql")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		_, err := Parse("", line)
		assert.Error(t, err, "zql: %q", line)
	}
}
