// +build !windows

package iosrc

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssue2234(t *testing.T) {
	u, err := ParseURI("http.06:28:42-07:00:00.log.gz")
	require.NoError(t, err)
	assert.Equal(t, FileScheme, u.Scheme)
	assert.Equal(t, "http.06:28:42-07:00:00.log.gz", path.Base(u.Path))
}
