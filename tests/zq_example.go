package tests

/*
Find valid ZQ examples in markdown, run them against
https://github.com/brimdata/zed-sample-data/zeek-default, and compare results in
docs with results produced.

Use markers in markdown fenced code blocks to denote either a zq command or
output from zq. Use is like:

```zq-command
zq "* | count()
```
```zq-output
1234
```

This is in compliance with https://spec.commonmark.org/0.29/#example-113

Doc authors MUST pair off each command and output in their own fenced code blocks.

A zq-command code block MUST be one line after the leading line, which includes
the info string. A zq-command MUST start with "zq". A zq-command MUST quote the
full zql with single quotes. The zql MAY contain double quotes, but it MUST NOT
contain single quotes.

Examples:

zql '* | count()' *.log.gz  # ok
zql  * | count()  *.log.gz  # not ok
zql "* | count()" *.log.gz  # not ok

zql 'field="value"   | count()' *.log.gz  # ok
zql 'field=\'value\' | count()' *.log.gz  # not ok
zql 'field='value'   | count()' *.log.gz  # not ok
zql "field=\"value\" | count()" *.log.gz  # not ok

A zq-command MUST reference one or more files or globs, expanded at
zed-sample-data/zeek-default.

Examples:

zql '* | count()' *.log.gz                # ok
zql '* | count()' conn.log.gz             # ok
zql '* | count()' conn.log.gz http.log.gz # ok
zql '* | count()' c*.log.gz d*.log.gz     # ok
zql '* | count()'                         # not ok

A zq-command MAY contain a sh-compliant comment string (denoted by '#') on the
line. Everything including and after the first # is stripped away.

A zq-output fenced code block MAY be multiple lines. zq-output MUST be verbatim
from the actual zq output.

zq-output MAY contain an optional marker to support record truncation. The
marker is denoted by "head:N" where N MUST be a non-negative integer
representing the number of lines to show. The marker MAY contain an ellipsis
via three dots "..." at the end to imply to readers the continuation of records
not shown.

Example:

```zq-output head:4
_PATH COUNT
conn  3
dhcp  2
dns   1
...
```

If head is malformed or N is invalid, fall back to verification against all
records.
*/

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
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
	Command         []string
	Expected        string
	OutputLineCount int
}

// Run runs a zq command and returns its output.
func (t *ZQExampleTest) Run() (string, error) {
	c := exec.Command(t.Command[0], t.Command[1:]...)
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

// QualifyCommand translates a zq-command example to a runnable command,
// including abspath to zq binary and globs turned into absolute file paths.
func QualifyCommand(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	command = strings.Split(command, "#")[0]

	pieces := strings.Split(command, "'")
	if len(pieces) != 3 {
		return nil, fmt.Errorf("could not split zq command 3 tokens: %s", command)
	}

	command_and_flags := strings.Split(strings.TrimSpace(pieces[0]), " ")
	if command_and_flags[0] != "zq" {
		return nil, fmt.Errorf("command does not start with zq: %s", command)
	}
	// Nice, but this makes unit testing more complicated.
	zq, err := ZQAbsPath()
	if err != nil {
		return nil, err
	}
	command_and_flags[0] = zq

	zql := strings.TrimSpace(pieces[1])

	var fileargs []string
	sampledata, err := ZedSampleDataAbsPath()
	if err != nil {
		return nil, err
	}
	for _, relglobarg := range strings.Split(strings.TrimSpace(pieces[2]), " ") {
		files, err := filepath.Glob(filepath.Join(sampledata, "zeek-default", relglobarg))
		if err != nil {
			return nil, err
		}
		fileargs = append(fileargs, files...)
	}

	finalized := command_and_flags
	finalized = append(finalized, zql)
	finalized = append(finalized, fileargs...)
	return finalized, nil
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
	source, err := ioutil.ReadFile(absfilename)
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
	for _, example := range examples {
		linenum := bytes.Count(source[:example.command.Info.Segment.Start], []byte("\n")) + 2
		testname := strings.TrimPrefix(absfilename, repopath) + ":" + strconv.Itoa(linenum)

		command, err := QualifyCommand(BlockString(example.command, source))
		if err != nil {
			return tests, err
		}

		output := strings.TrimSuffix(BlockString(example.output, source), "...\n")

		tests = append(tests, ZQExampleTest{testname, command, output, example.outputLineCount})
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
