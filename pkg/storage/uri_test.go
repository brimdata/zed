package storage

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURIStdio(t *testing.T) {
	u, err := ParseURI("stdio:stdout")
	require.NoError(t, err)
	assert.Equal(t, "stdio:stdout", u.String())
	u2, err := ParseURI(u.String())
	require.NoError(t, err)
	assert.Equal(t, u, u2)
}

func TestURIRelative(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)
	dir = filepath.ToSlash(dir)
	// This case is for windows only.
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	expected := "file://" + path.Join(dir, "relative", "path")

	u1, err := ParseURI("relative/path")
	require.NoError(t, err)
	assert.Equal(t, expected, u1.String())
}

type jsonStruct struct{ Test *URI }

func TestURIJSON(t *testing.T) {
	expected := "s3://test-bucket/test/key"
	u, err := ParseURI(expected)
	require.NoError(t, err)
	d, err := json.Marshal(jsonStruct{u})
	require.NoError(t, err)
	var out jsonStruct
	require.NoError(t, json.Unmarshal(d, &out))
	assert.Equal(t, expected, out.Test.String())
}

func TestURIParseEmpty(t *testing.T) {
	u, err := ParseURI("")
	require.NoError(t, err)
	assert.Equal(t, u, &URI{})
	assert.True(t, u.IsZero())
}

func TestURISerializeEmpty(t *testing.T) {
	var u URI
	assert.Equal(t, "", u.String())
}

func TestPathWithEncodedChars(t *testing.T) {
	// Create a real directory since ParseURI will always return an absolute
	// path, and this will verify Windows path handling as well.
	p := filepath.Join(t.TempDir(), "file with spaces")
	u, err := ParseURI(p)
	require.NoError(t, err)
	assert.Equal(t, p, u.Filepath())
}
