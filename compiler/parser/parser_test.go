package parser_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/ztest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchForZed() ([]string, error) {
	var zed []string
	pattern := fmt.Sprintf(`.*ztests\%c.*\.yaml$`, filepath.Separator)
	re := regexp.MustCompile(pattern)
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") && re.MatchString(path) {
			zt, err := ztest.FromYAMLFile(path)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			z := zt.Zed
			if z == "" || z == "*" {
				return nil
			}
			zed = append(zed, z)
		}
		return err
	})
	return zed, err
}

func parsePEGjs(z string) ([]byte, error) {
	cmd := exec.Command("node", "run.js", "-e", "start")
	cmd.Stdin = strings.NewReader(z)
	return cmd.Output()
}

func parseProc(z string) ([]byte, error) {
	proc, err := compiler.ParseProc(z)
	if err != nil {
		return nil, err
	}
	return json.Marshal(proc)
}

func parsePigeon(z string) ([]byte, error) {
	ast, err := parser.Parse("", []byte(z))
	if err != nil {
		return nil, err
	}
	return json.Marshal(ast)
}

// testZed parses the Zed query in line by both the Go and Javascript
// parsers.  It checks both that the parse is successful and that the
// two resulting ASTs are equivalent.  On the go side, we take a round
// trip through json marshal and unmarshal to turn the parse-tree types
// into generic JSON types.
func testZed(t *testing.T, line string) {
	pigeonJSON, err := parsePigeon(line)
	assert.NoError(t, err, "parsePigeon: %q", line)

	astJSON, err := parseProc(line)
	assert.NoError(t, err, "parseProc: %q", line)

	assert.JSONEq(t, string(pigeonJSON), string(astJSON), "pigeon and ast.Proc mismatch: %q", line)

	if runtime.GOOS != "windows" {
		pegJSON, err := parsePEGjs(line)
		assert.NoError(t, err, "parsePEGjs: %q", line)
		assert.JSONEq(t, string(pigeonJSON), string(pegJSON), "pigeon and PEGjs mismatch: %q", line)
	}
}

func TestValid(t *testing.T) {
	file, err := fs.Open("valid.zed")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		testZed(t, string(line))
	}
}

func TestZtestZed(t *testing.T) {
	zed, err := searchForZed()
	require.NoError(t, err)
	for _, z := range zed {
		testZed(t, z)
	}
}

func TestInvalid(t *testing.T) {
	file, err := fs.Open("invalid.zed")
	require.NoError(t, err)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		_, err := parser.Parse("", line)
		assert.Error(t, err, "Zed: %q", line)
	}
}
