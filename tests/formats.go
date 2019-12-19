package tests

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mccanne/zq/pkg/test"
)

type filespec struct {
	path      string
	base      string
	direction string
	ext       string
	format    string
}

func match(subdir, name, direction string) *filespec {
	components := strings.Split(name, ".")
	if len(components) != 3 {
		//XXX warning
		return nil
	}
	if components[1] != direction {
		return nil
	}
	var format string
	ext := components[2]
	switch ext {
	case "log":
		format = "zeek"
	case "zng":
		format = "zng"
	case "zjson":
		format = "zjson"
	case "bzng":
		format = "bzng"
	case "json", "ndjson":
		format = "ndjson"
	case "txt", "text":
		format = "text"
	case "tbl", "table":
		format = "table"
	default:
		return nil
	}
	return &filespec{
		path:      filepath.Join(subdir, name),
		base:      components[0],
		direction: direction,
		ext:       ext,
		format:    format,
	}
}

func findMatch(subdir string, entries []os.FileInfo, spec filespec) *filespec {
	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		out := match(subdir, f.Name(), "out")
		if out != nil && out.base == spec.base {
			return out
		}
	}
	return nil
}

func findFiles(entries []os.FileInfo, subdir, direction string) []filespec {
	var out []filespec
	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		// name.dir.ext
		s := match(subdir, f.Name(), direction)
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

func findTestDir(out []test.Exec, subdir string) ([]test.Exec, error) {
	entries, err := ioutil.ReadDir(subdir)
	if err != nil {
		return nil, err
	}
	inputs := findFiles(entries, subdir, "in")
	for _, input := range inputs {
		output := findMatch(subdir, entries, input)
		if output == nil {
			continue
		}
		inbytes, err := ioutil.ReadFile(input.path)
		if err != nil {
			return nil, err
		}
		outbytes, err := ioutil.ReadFile(output.path)
		if err != nil {
			return nil, err
		}
		cmd := test.Exec{
			Name:     filepath.Join(subdir, input.base),
			Command:  "zq -f " + output.format + " -",
			Input:    string(inbytes),
			Expected: string(outbytes),
		}
		out = append(out, cmd)
	}
	return out, nil
}

func findTests(dir string) ([]test.Exec, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []test.Exec
	for _, subdir := range entries {
		if subdir.IsDir() {
			path := filepath.Join(dir, subdir.Name())
			out, err = findTestDir(out, path)
			if err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
