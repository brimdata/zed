package archive

import (
	"context"
	"io"
	"strings"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/options"
	"github.com/brimsec/zq/zng/resolver"
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

type multiSource struct {
	ark   *Archive
	paths []string
}

// NewMultiSource returns a driver.MultiSource for an Archive. If no paths are
// specified, the MultiSource will send a source for each chunk file, and
// report the same ordering as the archive. Otherwise, the sources come from
// localizing the given paths to each chunk's directory (recognizing "_" as the
// chunk file itself), with no defined ordering.
func NewMultiSource(ark *Archive, paths []string) driver.MultiSource {
	if len(paths) == 0 {
		paths = []string{"_"}
	}
	return &multiSource{
		ark:   ark,
		paths: paths,
	}
}

func (ams *multiSource) OrderInfo() (string, bool) {
	if len(ams.paths) == 1 && ams.paths[0] == "_" {
		return "ts", ams.ark.DataSortDirection == zbuf.DirTimeReverse
	}
	return "", false
}

type archiveSource struct {
	scanner.Scanner
	io.Closer
}

func (ams *multiSource) SendSources(ctx context.Context, zctx *resolver.Context, sf driver.SourceFilter, srcChan chan driver.SourceOpener) error {
	return SpanWalk(ams.ark, func(si SpanInfo, zardir iosrc.URI) error {
		if !sf.Span.Overlaps(si.Span) {
			return nil
		}
		so := func() (driver.ScannerCloser, error) {
			// In the future, we could determine if any microindex in
			// this zardir would be useful as a filter by comparing the
			// filter expression in sf.FilterExpr against the available
			// indices, then run a Find against the index to avoid reading
			// the entire chunk.
			var paths []string
			for _, input := range ams.paths {
				p := Localize(zardir, input)
				// XXX Detector doesn't support file uri's.
				if p.Scheme == "file" {
					paths = append(paths, p.Filepath())
				} else {
					paths = append(paths, p.String())
				}
			}
			rc := detector.MultiFileReader(zctx, paths, options.Reader{Format: "zng"})
			sn, err := scanner.NewScanner(ctx, rc, sf.Filter, sf.FilterExpr, sf.Span)
			if err != nil {
				return nil, err
			}
			return &archiveSource{Scanner: sn, Closer: rc}, nil
		}
		select {
		case srcChan <- so:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}
