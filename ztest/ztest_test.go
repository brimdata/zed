package ztest

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkip(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, "script test on Windows", (&ZTest{Script: "x"}).ShouldSkip(""))
	} else {
		assert.Equal(t, "script test on in-process run", (&ZTest{Script: "x"}).ShouldSkip(""))
	}
	assert.Equal(t, "reason", (&ZTest{Skip: "reason"}).ShouldSkip(""))
	assert.Equal(t, `tag "x" does not match ZTEST_TAG=""`, (&ZTest{Tag: "x"}).ShouldSkip(""))
}

func TestRunScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test on Windows because RunScript uses cmd.exe instead of bash")
	}
	t.Run("success", func(t *testing.T) {
		stdout := "1\n"
		stderr := "2\n"
		err := (&ZTest{
			Script: "echo 1; echo 2 >&2",
			Outputs: []File{
				{Name: "stdout", Data: &stdout},
				{Name: "stderr", Data: &stderr},
			},
		}).RunScript("", "", "")
		assert.NoError(t, err)
	})
	t.Run("error", func(t *testing.T) {
		err := (&ZTest{
			Script:  "echo 1; echo 2 >&2; exit 3",
			Outputs: []File{},
		}).RunScript("", "", "")
		assert.EqualError(t, err, "script failed: exit status 3\n=== stdout ===\n1\n=== stderr ===\n2\n")
	})
}
