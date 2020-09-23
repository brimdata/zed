package zql

import (
	"bufio"
	"fmt"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	file, err := fs.Open("valid.zql")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		_, err := Parse("", line)
		assert.NoError(t, err, "zql: %q", line)
	}
}

func TestInvalid(t *testing.T) {
	file, err := fs.Open("invalid.zql")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		_, err := Parse("", line)
		assert.Error(t, err, "zql: %q", line)
	}
}

// parseString is a helper for testing string parsing.  It wraps the
// given string in a simple zql query, parses it, and extracts the parsed
// string from inside the AST.
func parseString(in string) (string, error) {
	code := fmt.Sprintf("s = \"%s\"", in)
	tree, err := ParseProc(code)
	if err != nil {
		return "", err
	}
	filt, ok := tree.(*ast.FilterProc)
	if !ok {
		return "", fmt.Errorf("Expected FilterProc got %T", tree)
	}
	comp, ok := filt.Filter.(*ast.CompareField)
	if !ok {
		return "", fmt.Errorf("Expected CompareField got %T", filt.Filter)
	}
	return comp.Value.Value, nil
}

// Test handling of unicode escapes in the parser
func TestUnicode(t *testing.T) {
	result, err := parseString("Sacr\u00e9 bleu!")
	assert.NoError(t, err, "Parse of string succeeded")
	assert.Equal(t, result, "Sacré bleu!", "Unicode escape without brackets parsed correctly")

	result, err = parseString("I love \\u{1F32E}s")
	assert.NoError(t, err, "Parse of string succeeded")
	assert.Equal(t, result, "I love 🌮s", "Unicode escape with brackets parsed correctly")
}
