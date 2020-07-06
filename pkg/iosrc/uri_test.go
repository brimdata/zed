package iosrc

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURIStdio(t *testing.T) {
	u, err := ParseURI("stdout")
	require.NoError(t, err)
	assert.Equal(t, "stdio:///stdout", u.String())
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
