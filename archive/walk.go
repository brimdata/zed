package archive

import (
	"os"
	"path/filepath"
	"strings"
)

func ZarDirToLog(path string) string {
	return strings.TrimSuffix(path, zarExt)
}

func LogToZarDir(path string) string {
	return path + zarExt
}

// Localize maps the provided slice of relative pathnames into
// absolute path names relative to the given zardir and returns
// the new pathnames as a slice.  The special name "_" is mapped
// to the path of the log file that corresponds to this zardir.
func Localize(zardir string, filenames []string) []string {
	var out []string
	for _, filename := range filenames {
		var s string
		if filename == "_" {
			s = ZarDirToLog(zardir)
		} else {
			s = filepath.Join(zardir, filename)
		}
		out = append(out, s)
	}
	return out
}

type Visitor func(zardir string) error

// Walk traverses the archive invoking the visitor on the zar dir corresponding
// to each log file, creating the zar dir if needed.
func Walk(ark *Archive, visit Visitor) error {
	return SpanWalk(ark, func(_ SpanInfo, zardir string) error {
		return visit(zardir)
	})
}

type SpanVisitor func(si SpanInfo, zardir string) error

func SpanWalk(ark *Archive, v SpanVisitor) error {
	for _, s := range ark.Meta.Spans {
		zardir := LogToZarDir(s.LogID.Path(ark))
		if err := os.MkdirAll(zardir, 0700); err != nil {
			return err
		}
		if err := v(s, zardir); err != nil {
			return err
		}
	}
	return nil
}

// RmDirs descends a directory hierarchy looking for zar dirs and remove
// each such directory and all of its contents.
func RmDirs(ark *Archive) error {
	return Walk(ark, os.RemoveAll)
}
