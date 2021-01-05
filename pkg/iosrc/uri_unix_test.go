// +build !windows

package iosrc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// github.com/brimsec/brim#1284
func TestURIUNCPath(t *testing.T) {
	uri, err := ParseURI(`//34.82.284.241/foo`)
	require.NoError(t, err)
	assert.Equal(t, "file://34.82.284.241/foo", uri.String())
	assert.Equal(t, `//34.82.284.241/foo`, uri.Filepath())
}
