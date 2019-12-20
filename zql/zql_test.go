package zql_test

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	zqlgo "github.com/mccanne/zq/zql"
	"github.com/mccanne/zq/zql/execjs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getJsPath() string {
	dir, _ := os.Getwd()
	return filepath.Join(dir, "../js/exec.js")
}

func TestValid(t *testing.T) {
	zqljs := execjs.Runner(getJsPath())
	file, err := os.Open("valid.zql")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		t.Run(line, func(t *testing.T) {
			goproc, err := zqlgo.ParseProc(line)
			assert.NoError(t, err, "zql (go): %q", line)
			jsproc, err := zqljs.ParseProc(line)
			assert.NoError(t, err, "zql (js): %q", line)
			assert.Equal(t, goproc, jsproc, "go and js parser output differs: %q", line)
		})
	}
}

func TestInvalid(t *testing.T) {
	zqljs := execjs.Runner(getJsPath())
	file, err := os.Open("invalid.zql")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		t.Run(line, func(t *testing.T) {
			_, err := zqlgo.ParseProc(line)
			assert.Error(t, err, "zql (go): %q", line)
			_, err = zqljs.ParseProc(line)
			assert.Error(t, err, "zql (js): %q", line)
		})
	}
}
