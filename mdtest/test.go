package mdtest

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// Test represents a single test in a Markdown file.
type Test struct {
	Command  string
	Dir      string
	Expected string
	Head     bool
	Line     int
}

// Run runs the test, returning nil on success.
func (t *Test) Run() error {
	c := exec.Command("bash", "-e", "-o", "pipefail")
	c.Dir = t.Dir
	c.Stdin = strings.NewReader(t.Command)
	outBytes, err := c.CombinedOutput()
	out := string(outBytes)
	if err != nil {
		if out != "" {
			return fmt.Errorf("%w\noutput:\n%s", err, out)
		}
		return err
	}
	if t.Head && len(out) > len(t.Expected) {
		out = out[:len(t.Expected)]
	}
	if out != t.Expected {
		diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(t.Expected),
			FromFile: "expected",
			B:        difflib.SplitLines(out),
			ToFile:   "actual",
			Context:  5,
		})
		if err != nil {
			return err
		}
		return fmt.Errorf("expected and actual output differ:\n%s", diff)
	}
	return nil
}
