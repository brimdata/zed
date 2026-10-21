package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	expected := filepath.Join(dir, "relative", "path")

	u1, err := ParseURI("relative/path")
	require.NoError(t, err)
	assert.Equal(t, expected, u1.String())
}

func TestURIFileString(t *testing.T) {
	// A file URI given with a scheme still prints as a plain path.
	u, err := ParseURI("file:///databases/test")
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("/databases/test"), u.String())
	// A file URI with a host isn't a local path so it keeps its URL form.
	u, err = ParseURI("file://host/databases/test")
	require.NoError(t, err)
	assert.Equal(t, "file://host/databases/test", u.String())
}

func TestURIFileStringRoundTrip(t *testing.T) {
	// Characters that are special in URLs must survive String and ParseURI
	// unescaped.
	p := filepath.Join(t.TempDir(), "a b", "%?#")
	u, err := ParseURI(p)
	require.NoError(t, err)
	assert.Equal(t, p, u.String())
	u2, err := ParseURI(u.String())
	require.NoError(t, err)
	assert.Equal(t, u, u2)
	b, err := u.MarshalText()
	require.NoError(t, err)
	var u3 URI
	require.NoError(t, u3.UnmarshalText(b))
	assert.Equal(t, *u, u3)
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

func TestParseURIWithBarePathContainingSpecialChars(t *testing.T) {
	// Create a real directory since ParseURI will always return an absolute
	// path, and this will verify Windows path handling as well.
	dir := t.TempDir()
	// "%" is special because it introduces a percent-encoded octet.  "?"
	// and "#" are special because they introduce a path's query component
	// and fragment identifier component, respectively.
	p := filepath.Join(dir, "%?#")
	u, err := ParseURI(p)
	require.NoError(t, err)
	assert.Equal(t, p, u.Filepath())
}
