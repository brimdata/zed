// Package mdtest finds example shell commands in Markdown files and runs them,
// checking for expected output.
//
// Example inputs, commands, and outputs are specified in fenced code blocks
// whose info string (https://spec.commonmark.org/0.29/#info-string) has
// mdtest-input, mdtest-command, or mdtest-output as the first word.  The
// mdtest-command and mdtest-output blocks must be paired.
//
//    ```mdtest-input file.txt
//    hello
//    ```
//    ```mdtest-command [path]
//    cat file.txt
//    ```
//    ```mdtest-output
//    hello
//    ```
//
// The content of each mdtest-command block is fed to "bash -e -o pipefail" on
// standard input.  The shell's working directory is a temporary directory
// populated with files described by any mdtest-input blocks in the same
// Markdown file and shared by other tests in the same file.  Alternatively, if
// the mdtest-command block's info string contains a second word, it specifies
// the shell's working directory as a path relative to the repository root, and
// files desribed by mdtest-input blocks are not available.  In either case, the
// shell's combined standard output and standard error must exactly match the
// content of the following mdtest-output block except as described below.
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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// File represents a Markdown file and the tests it contains.
type File struct {
	Path   string
	Inputs map[string]string
	Tests  []*Test
}

// Load walks the file tree rooted at the current working directory, looking for
// Markdown files containing tests.  Any file whose name ends with ".md" is
// considered a Markdown file.  Files containing no tests are ignored.
func Load() ([]*File, error) {
	var files []*File
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		inputs, tests, err := parseMarkdown(b)
		if err != nil {
			var le lineError
			if errors.As(err, &le) {
				return fmt.Errorf("%s:%d: %s", path, le.line, le.msg)
			}
			return fmt.Errorf("%s: %w", path, err)
		}
		if len(tests) > 0 {
			files = append(files, &File{
				Path:   path,
				Inputs: inputs,
				Tests:  tests,
			})
		}
		return nil
	})
	return files, err
}

// Run runs the file's tests.  It runs relative-directory-style tests (Test.Dir
// != "") in parallel, with the shell working directory set to Test.Dir, and it
// runs temporary-directory-style tests (Test.Dir == "") sequentially, with the
// shell working directory set to a shared temporary directory.
func (f *File) Run(t *testing.T) {
	tempdir := t.TempDir()
	for filename, content := range f.Inputs {
		if err := os.WriteFile(filepath.Join(tempdir, filename), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range f.Tests {
		// Copy struct so assignment to tt.Dir below won't modify f.Tests.
		tt := *tt
		t.Run(strconv.Itoa(tt.Line), func(t *testing.T) {
			if tt.Dir == "" {
				tt.Dir = tempdir
			} else {
				t.Parallel()
			}
			if err := tt.Run(); err != nil {
				// Lead with newline so line-numbered errors are
				// navigable in editors.
				t.Fatalf("\n%s:%d: %s", f.Path, tt.Line, err)
			}
		})
	}
}

func parseMarkdown(source []byte) (map[string]string, []*Test, error) {
	var commandFCB *ast.FencedCodeBlock
	var inputs map[string]string
	var tests []*Test
	doc := goldmark.DefaultParser().Parse(text.NewReader(source))
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		fcb, ok := n.(*ast.FencedCodeBlock)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		switch string(fcb.Language(source)) {
		case "mdtest-input":
			words := fcbInfoWords(fcb, source)
			if len(words) < 2 {
				return ast.WalkStop, errors.New("mdtest-input without file name")
			}
			filename := words[1]
			if inputs == nil {
				inputs = map[string]string{}
			}
			if _, ok := inputs[filename]; ok {
				return ast.WalkStop, errors.New("mdtest-input with duplicate file name")
			}
			inputs[filename] = fcbLines(fcb, source)
		case "mdtest-command":
			if commandFCB != nil {
				return ast.WalkStop, fcbError(commandFCB, source, "unpaired mdtest-command")
			}
			commandFCB = fcb
		case "mdtest-output":
			if commandFCB == nil {
				return ast.WalkStop, fcbError(fcb, source, "unpaired mdtest-output")
			}
			var dir string
			if words := fcbInfoWords(commandFCB, source); len(words) > 1 {
				dir = words[1]
			}
			var head bool
			if words := fcbInfoWords(fcb, source); len(words) > 1 && words[1] == "head" {
				head = true
			}
			tests = append(tests, &Test{
				Command:  fcbLines(commandFCB, source),
				Dir:      dir,
				Expected: strings.TrimSuffix(fcbLines(fcb, source), "...\n"),
				Head:     head,
				Line:     fcbLineNumber(commandFCB, source),
			})
			commandFCB = nil
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, nil, err
	}
	if commandFCB != nil {
		return nil, nil, fcbError(commandFCB, source, "unpaired mdtest-command")
	}
	return inputs, tests, nil
}

func fcbError(fcb *ast.FencedCodeBlock, source []byte, msg string) error {
	return lineError{line: fcbLineNumber(fcb, source), msg: msg}
}

func fcbInfoWords(fcb *ast.FencedCodeBlock, source []byte) []string {
	return strings.Fields(string(fcb.Info.Segment.Value(source)))
}

func fcbLineNumber(fcb *ast.FencedCodeBlock, source []byte) int {
	return bytes.Count(source[:fcb.Info.Segment.Start], []byte("\n")) + 1
}

func fcbLines(fcb *ast.FencedCodeBlock, source []byte) string {
	var b strings.Builder
	segments := fcb.Lines()
	for _, s := range segments.Sliced(0, segments.Len()) {
		b.Write(s.Value(source))
	}
	return b.String()
}

type lineError struct {
	line int
	msg  string
}

func (l lineError) Error() string {
	return fmt.Sprintf("line %d: %s", l.line, l.msg)
}
