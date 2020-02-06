package tests

/*
Find valid ZQ examples in markdown, run them against
https://github.com/brimsec/zq-sample-data/zeek-default, and compare results in
docs with results produced.

In separate patches:
- Deal with examples that need head
- Find files as opposed to hard-coding them

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
zq-sample-data/zeek-default.

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

TODO: Support doc authors' desire to request and receive head(1) semantics
without including head as a proc. Related, also support ellipsis on the last
line to allow a doc author to convey continuation of records not shown. The
CommonMark spec allows this. Github supports the CommonMark spec, and goldmark
passes all CommonMark tests.

Proposed syntax Example:

```zq-command head:10
zq -f table "* | count() by query" dns.log.gz
```
```zq-output
QUERY                                                     COUNT
-                                                         2
goo                                                       20
t.co                                                      8
tmsc                                                      70
da.gd                                                     4
local                                                     12
bit.ly                                                    10
goo.gl                                                    4
(empty)                                                   12
...
```
... and the result still be correct. There are actually over 1000 lines when
actually running this command.

See https://gist.github.com/mikesbrown/f77cb939a43f80f2e019afba212c8c05 for how
Github shows these.
*/

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type ZQExampleBlockType string

const (
	ZQCommand ZQExampleBlockType = "zq-command"
	ZQOutput  ZQExampleBlockType = "zq-output"
)

// ZQExamplePair holds a ZQ example as found in markdown.
type ZQExamplePair struct {
	command *ast.FencedCodeBlock
	output  *ast.FencedCodeBlock
}

// ZQExampleTest holds a ZQ example as a testcase found from mardown, derived
// from a ZQExamplePair.
type ZQExampleTest struct {
	Name     string
	Command  []string
	Expected string
}

// CollectExamples returns a zq-command / zq-output pairs from a single
// markdown source after parsing it as a goldmark AST.
func CollectExamples(node ast.Node, source []byte) ([]ZQExamplePair, error) {
	var examples []ZQExamplePair
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
			examples = append(examples, ZQExamplePair{command, fcb})
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
	sampledata, err := ZQSampleDataAbsPath()
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
	var examples []ZQExamplePair
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
	for i, example := range examples {
		// Convert strings like
		// /home/user/zql/docs/processors/README.md to
		// /zql/docs/processors/README.md . RepoAbsPath() does not
		// include a trailing filepath.Separator in its return.
		testname := strings.TrimPrefix(absfilename, repopath)
		// Now convert strings like /zql/docs/processors/README.md to
		// zql/docs/processors/README.md1 go test will call such a test
		// something like
		// TestMarkdownExamples/zql/docs/processors/README.md1
		testname = strings.TrimPrefix(testname, string(filepath.Separator)) + strconv.Itoa(i+1)

		command, err := QualifyCommand(BlockString(example.command, source))
		if err != nil {
			return tests, err
		}

		output := strings.TrimSuffix(BlockString(example.output, source), "...\n")

		tests = append(tests, ZQExampleTest{testname, command, output})
	}
	return tests, nil
}

// DocMarkdownFiles returns markdown files to inspect.
func DocMarkdownFiles() ([]string, error) {
	// This needs to find markdown files in the repo. Right now we just
	// declare them directly.
	files := []string{
		"zql/docs/processors/README.md",
		"zql/docs/search-syntax/README.md",
	}
	repopath, err := RepoAbsPath()
	if err != nil {
		return nil, err
	}
	for i, file := range files {
		files[i] = filepath.Join(repopath, file)
	}
	return files, nil
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
