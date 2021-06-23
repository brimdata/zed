// Package mdtest finds example shell commands in Markdown files and runs them,
// checking for expected output.
//
// Example commands and outputs are specified in fenced code blocks whose info
// string (https://spec.commonmark.org/0.29/#info-string) has mdtest-command or
// mdtest-output as the first word.  These blocks must be paired.
//
//    ```mdtest-command [path]
//    echo hello
//    ```
//    ```mdtest-output
//    hello
//    ```
//
// The content of each mdtest-command block is fed to "bash -e -o pipefail" on
// standard input.  The shell's working directory is the repository root unless
// the block's info string contains a second word, which then specifies the
// working directory as a path relative to the repository root.  The shell's
// combined standard output and standard error must exactly match the content of
// the following mdtest-output block except as described below.
//
// If head appears as the second word in an mdtest-output block's info string,
// then any "...\n" suffix of the block content is ignored, and what remains
// must be a prefix of the shell output.
//
//    ```mdtest-command
//    echo hello
//    echo goodbye
//    ```
//    ```mdtest-output head
//    hello
//    ...
//    ```
package mdtest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type ZQExampleBlockType string

const (
	ZQCommand ZQExampleBlockType = "mdtest-command"
	ZQOutput  ZQExampleBlockType = "mdtest-output"
)

// ZQExampleInfo holds a ZQ example as found in markdown.
type ZQExampleInfo struct {
	command *ast.FencedCodeBlock
	output  *ast.FencedCodeBlock
}

// ZQExampleTest holds a ZQ example as a testcase found from mardown, derived
// from a ZQExampleInfo.
type ZQExampleTest struct {
	Name     string
	Command  string
	Dir      string
	Expected string
	Head     bool
}

// Run runs a zq command and returns its output.
func (t *ZQExampleTest) Run() (string, error) {
	c := exec.Command("bash", "-e", "-o", "pipefail")
	c.Dir = t.Dir
	c.Stdin = strings.NewReader(t.Command)
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	err := c.Run()
	out := b.String()
	if t.Head && len(out) > len(t.Expected) {
		out = out[:len(t.Expected)]
	}
	return out, err
}

// CollectExamples returns mdtest-command / zq-output pairs from a single
// markdown source after parsing it as a goldmark AST.
func CollectExamples(node ast.Node, source []byte) ([]ZQExampleInfo, error) {
	var examples []ZQExampleInfo
	var command *ast.FencedCodeBlock

	err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// Walk() calls its walker func twice. Once when entering and
		// once before exiting, after walking any children. We need
		// only do this processing once.
		if !entering || n == nil || n.Kind() != ast.KindFencedCodeBlock {
			return ast.WalkContinue, nil
		}

		fcb, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkStop,
				fmt.Errorf("likely goldmark bug: Kind() reports a " +
					"FencedCodeBlock, but the type assertion failed")
		}
		bt := ZQExampleBlockType(fcb.Language(source))
		switch bt {
		case ZQCommand:
			if command != nil {
				return ast.WalkStop,
					fmt.Errorf("subsequent %s after another %s", bt, ZQCommand)
			}
			command = fcb
		case ZQOutput:
			if command == nil {
				return ast.WalkStop,
					fmt.Errorf("%s without a preceeding %s", bt, ZQCommand)
			}
			examples = append(examples, ZQExampleInfo{command, fcb})
			command = nil
			// A fenced code block need not specify an info string, or it
			// could be arbitrary. The default case is to ignore everything
			// else.
		}
		return ast.WalkContinue, nil
	})

	if command != nil && err == nil {
		err = fmt.Errorf("%s without a following %s", ZQCommand, ZQOutput)
	}
	return examples, err
}

// BlockString returns the text of a ast.FencedCodeBlock as a string.
func BlockString(fcb *ast.FencedCodeBlock, source []byte) string {
	var b strings.Builder
	for i := 0; i < fcb.Lines().Len(); i++ {
		line := fcb.Lines().At(i)
		b.Write(line.Value(source))
	}
	return b.String()
}

// TestcasesFromFile returns ZQ example test cases from ZQ example pairs found
// in a file.
func TestcasesFromFile(filename string) ([]ZQExampleTest, error) {
	var tests []ZQExampleTest
	var examples []ZQExampleInfo
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	reader := text.NewReader(source)
	parser := goldmark.DefaultParser()
	doc := parser.Parse(reader)
	examples, err = CollectExamples(doc, source)
	if err != nil {
		return nil, err
	}
	for _, e := range examples {
		linenum := bytes.Count(source[:e.command.Info.Segment.Start], []byte("\n")) + 2
		var commandDir string
		if words := fcbInfoWords(e.command, source); len(words) > 1 {
			commandDir = words[1]
		}
		var head bool
		if words := fcbInfoWords(e.output, source); len(words) > 1 && words[1] == "head" {
			head = true
		}
		tests = append(tests, ZQExampleTest{
			Name:     filename + ":" + strconv.Itoa(linenum),
			Command:  BlockString(e.command, source),
			Dir:      commandDir,
			Expected: strings.TrimSuffix(BlockString(e.output, source), "...\n"),
			Head:     head,
		})
	}
	return tests, nil
}

func fcbInfoWords(fcb *ast.FencedCodeBlock, source []byte) []string {
	return strings.Fields(string(fcb.Info.Segment.Value(source)))
}

// DocMarkdownFiles returns markdown files to inspect.
func DocMarkdownFiles() ([]string, error) {
	var files []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ZQExampleTestCases returns all test cases derived from doc examples.
func ZQExampleTestCases() ([]ZQExampleTest, error) {
	var alltests []ZQExampleTest
	files, err := DocMarkdownFiles()
	if err != nil {
		return nil, err
	}
	for _, filename := range files {
		tests, err := TestcasesFromFile(filename)
		if err != nil {
			return nil, err
		}
		alltests = append(alltests, tests...)
	}
	return alltests, nil
}
