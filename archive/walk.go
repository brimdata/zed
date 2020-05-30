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

// Localize maps the provided relative path name into absolute path
// names relative to the given zardir and returns the result.  The
// special name "_" is mapped to the path of the log file that
// corresponds to this zardir.
func Localize(zardir string, pathname string) string {
	if pathname == "_" {
		return ZarDirToLog(zardir)
	}
	return filepath.Join(zardir, pathname)
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
