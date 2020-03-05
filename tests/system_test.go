// +build system

package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const zqpath = "../dist"

func TestInternal(t *testing.T) {
	t.Parallel()
	for _, d := range internals {
		t.Run(d.Name, func(t *testing.T) {
			results, err := d.Run()
			assert.True(t, errors.Is(err, d.ExpectedErr), "expected %v error, got %v", d.ExpectedErr, err)
			assert.Exactly(t, d.Expected, results, "Wrong query results")
		})
	}
}

func TestCommands(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{})
	for _, cmd := range commands {
		name := cmd.Name
		if _, ok := seen[name]; ok {
			t.Logf("test %s: skipping extra (unique test names are required)", name)
		}
		seen[name] = struct{}{}
		t.Run(name, func(t *testing.T) {
			results, err := cmd.Run(zqpath)
			require.NoError(t, err)
			assert.Exactly(t, cmd.Expected, results, "Wrong command results")
		})
	}
}
