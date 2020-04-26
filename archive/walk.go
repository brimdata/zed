package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Visitor func(zardir string) error

func IsZarDir(path string) bool {
	return filepath.Ext(path) == zarExt
}

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

// Walk descends a directory hierarchy looking for zar directories and
// invokes the visitor on each log file with a zar dir, providing the path
// of the log file and path of the zar dir.
func Walk(dir string, visit Visitor) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%q: %v", path, err)
		}
		if info.IsDir() && IsZarDir(path) {
			if err := visit(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}
		// descend...
		return nil
	})
}

// MkDirs descends a directory hierarchy looking for file paths that match
// the provided regular expression and creates a zar directory for each
// such file path.
func MkDirs(dir string, re *regexp.Regexp) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%q: %v", path, err)
		}
		if info.IsDir() {
			if IsZarDir(path) {
				// don't create zar dirs inside of existing
				// zar dirs
				return filepath.SkipDir
			}
			// descend...
			return nil
		}
		if re.Match([]byte(path)) {
			zardir := LogToZarDir(path)
			if err := os.Mkdir(zardir, 0700); err != nil {
				if os.IsExist(err) {
					err = nil
				}
				return err
			}
		}
		return nil
	})
}

// RmDirs descends a directory hierarchy looking for zar dirs and remove
// each such directory and all of its contents.
func RmDirs(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%q: %v", path, err)
		}
		if info.IsDir() && IsZarDir(path) {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}
		// descend...
		return nil
	})
}
