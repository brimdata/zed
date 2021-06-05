// Package tests finds example shell commands in Markdown files and runs them,
// checking for expected output.
//
// Example commands and outputs are specified in fenced code blocks whose info
// string (https://spec.commonmark.org/0.29/#info-string) has zq-command or
// zq-output as the first word.  These blocks must be paired.
//
//    ```zq-command [path]
//    echo hello
//    ```
//    ```zq-output
//    hello
//    ```
//
// The content of each zq-command block is fed to "bash -e -o pipefail" on
// standard input.  The shell's working directory is the repository root unless
// the block's info string contains a second word, which then specifies the
// working directory as a path relative to the repository root.  The shell's
// combined standard output and standard error must exactly match the content of
// the following zq-output block except as described below.
//
// If head:N appears as the second word in a zq-output block's info string,
// where N is a non-negative interger, then only the first N lines of shell
// output are examined, and any "...\n" suffix of the block content is ignored.
//
//    ```zq-command
//    echo hello
//    echo goodbye
//    ```
//    ```zq-output head:1
//    hello
//    ...
//    ```
//
// If head is malformed or N is invalid, the word is ignored.
package tests

import (
	"bufio"
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
	ZQCommand    ZQExampleBlockType = "zq-command"
	ZQOutput     ZQExampleBlockType = "zq-output"
	ZQOutputHead string             = "head:"
)

// ZQExampleInfo holds a ZQ example as found in markdown.
type ZQExampleInfo struct {
	command         *ast.FencedCodeBlock
	output          *ast.FencedCodeBlock
	outputLineCount int
}

// ZQExampleTest holds a ZQ example as a testcase found from mardown, derived
// from a ZQExampleInfo.
type ZQExampleTest struct {
	Name            string
	Command         string
	Dir             string
	Expected        string
	OutputLineCount int
}

// Run runs a zq command and returns its output.
func (t *ZQExampleTest) Run() (string, error) {
	c := exec.Command("bash", "-e", "-o", "pipefail")
	c.Dir = t.Dir
	c.Stdin = strings.NewReader(t.Command)
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	if err := c.Run(); err != nil {
		return string(b.Bytes()), err
	}
	scanner := bufio.NewScanner(&b)
	i := 0
	var s string
	for scanner.Scan() {
		if i == t.OutputLineCount {
			break
		}
		s += scanner.Text() + "\n"
		i++
	}
	if err := scanner.Err(); err != nil {
		return s, err
	}
	return s, nil
}

// ZQOutputLineCount returns the number of lines against which zq-output should
// be verified.
func ZQOutputLineCount(fcb *ast.FencedCodeBlock, source []byte) int {
	count := fcb.Lines().Len()
	if fcb.Info == nil {
		return count
	}
	info := strings.Split(string(fcb.Info.Segment.Value(source)), ZQOutputHead)
	if len(info) != 2 {
		return count
	}
	customCount, err := strconv.Atoi(info[1])
	if err != nil || customCount < 0 {
		return count
	}
	return customCount
}

// CollectExamples returns zq-command / zq-output pairs from a single
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
			outputLineCount := ZQOutputLineCount(fcb, source)
			examples = append(examples, ZQExampleInfo{command, fcb, outputLineCount})
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
	absfilename, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	repopath, err := RepoAbsPath()
	if err != nil {
		return nil, err
	}
	source, err := os.ReadFile(absfilename)
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
	repopath += string(filepath.Separator)
	for _, e := range examples {
		linenum := bytes.Count(source[:e.command.Info.Segment.Start], []byte("\n")) + 2
		var commandDir string
		if infoWords := strings.Fields(string(e.command.Info.Segment.Value(source))); len(infoWords) > 1 {
			commandDir = infoWords[1]
		}
		tests = append(tests, ZQExampleTest{
			Name:            strings.TrimPrefix(absfilename, repopath) + ":" + strconv.Itoa(linenum),
			Command:         BlockString(e.command, source),
			Dir:             filepath.Join(repopath, commandDir),
			Expected:        strings.TrimSuffix(BlockString(e.output, source), "...\n"),
			OutputLineCount: e.outputLineCount,
		})
	}
	return tests, nil
}

// DocMarkdownFiles returns markdown files to inspect.
func DocMarkdownFiles() ([]string, error) {
	repopath, err := RepoAbsPath()
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.Walk(repopath, func(path string, info os.FileInfo, err error) error {
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
