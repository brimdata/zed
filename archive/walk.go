package archive

import (
	"strings"

	"github.com/brimsec/zq/pkg/iosrc"
)

func ZarDirToLog(uri iosrc.URI) iosrc.URI {
	uri.Path = strings.TrimSuffix(uri.Path, zarExt)
	return uri
}

func LogToZarDir(uri iosrc.URI) iosrc.URI {
	uri.Path = uri.Path + zarExt
	return uri
}

// Localize maps the provided relative path name into absolute path
// names relative to the given zardir and returns the result.  The
// special name "_" is mapped to the path of the log file that
// corresponds to this zardir.
func Localize(zardir iosrc.URI, pathname string) iosrc.URI {
	if pathname == "_" {
		return ZarDirToLog(zardir)
	}
	return zardir.AppendPath(pathname)
}

type Visitor func(zardir iosrc.URI) error

// Walk traverses the archive invoking the visitor on the zar dir corresponding
// to each log file, creating the zar dir if needed.
func Walk(ark *Archive, visit Visitor) error {
	return SpanWalk(ark, func(_ SpanInfo, zardir iosrc.URI) error {
		return visit(zardir)
	})
}

type SpanVisitor func(si SpanInfo, zardir iosrc.URI) error

func SpanWalk(ark *Archive, v SpanVisitor) error {
	if _, err := ark.UpdateCheck(); err != nil {
		return err
	}

	ark.mu.RLock()
	defer ark.mu.RUnlock()

	for _, s := range ark.spans {
		zardir := LogToZarDir(s.LogID.Path(ark))
		if dirmkr, ok := ark.dataSrc.(iosrc.DirMaker); ok {
			if err := dirmkr.MkdirAll(zardir, 0700); err != nil {
				return err
			}
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
	return Walk(ark, ark.dataSrc.RemoveAll)
}
