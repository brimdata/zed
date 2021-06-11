// +build heavy

package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownExamples(t *testing.T) {
	t.Parallel()
	alltests, err := ZQExampleTestCases()
	require.Equal(t, nil, err, "error loading test cases: %v", err)
	for _, exampletest := range alltests {
		exampletest := exampletest
		t.Run(exampletest.Name, func(t *testing.T) {
			t.Parallel()
			cmdOutput, err := exampletest.Run(t)
			require.Equal(t, nil, err, "error running command %v: %v", exampletest.Command, err)
			assert.Equal(t, exampletest.Expected, cmdOutput, "output mismatch with command %v", exampletest.Command)
		})
	}
}
