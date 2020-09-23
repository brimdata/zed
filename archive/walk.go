package archive

import (
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/byteconv"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/segmentio/ksuid"
)

const (
	dataDirname = "zd"
	zarExt      = ".zar"
)

// A fileKind is the first part of a file name, used to differentiate files
// when they are listed from the archive's backing store.
type fileKind string

const (
	fileKindUnknown fileKind = ""
	fileKindData             = "d"
	fileKindSeek             = "ts"
)

// A dataFile holds archive record data. Only one kind of data file
// currently exists, representing a file created during ingest.
type dataFile struct {
	id   ksuid.KSUID
	kind fileKind
}

func newDataFile() dataFile {
	return dataFile{ksuid.New(), fileKindData}
}

func (f dataFile) name() string {
	return fmt.Sprintf("%s-%s.zng", f.kind, f.id)
}

var dataFileNameRegex = regexp.MustCompile(`d-([0-9A-Za-z]{27}).zng$`)

func dataFileNameMatch(s string) (f dataFile, ok bool) {
	match := dataFileNameRegex.FindSubmatch([]byte(s))
	if match == nil {
		return
	}
	id, err := ksuid.Parse(byteconv.UnsafeString(match[1]))
	if err != nil {
		return
	}
	return dataFile{id, fileKindData}, true
}

// A seekIndexFile is a microindex whose keys are record timestamps, and whose
// values are seek offsets into the data file with the same uuid. The name
// of a seekIndexFile encodes the number of records, and the first & last
// record timestamps of the corresponding data file. The order of the
// first & last records matches the archive's data order.
type seekIndexFile struct {
	id          ksuid.KSUID
	first       nano.Ts
	last        nano.Ts
	recordCount int64
}

func (f seekIndexFile) name() string {
	return fmt.Sprintf("%s-%s-%d-%d-%d.zng", fileKindSeek, f.id, f.recordCount, f.first, f.last)
}

func (f seekIndexFile) span() nano.Span {
	return nano.Span{Ts: f.first, Dur: 1}.Union(nano.Span{Ts: f.last, Dur: 1})
}

var seekIndexNameRegex = regexp.MustCompile(`ts-([0-9A-Za-z]{27})-([0-9]+)-([0-9]+)-([0-9]+).zng$`)

func seekIndexNameMatch(s string) (f seekIndexFile, ok bool) {
	match := seekIndexNameRegex.FindSubmatch([]byte(s))
	if match == nil {
		return
	}
	id, err := ksuid.Parse(byteconv.UnsafeString(match[1]))
	if err != nil {
		return
	}
	recordCount, err := strconv.ParseInt(byteconv.UnsafeString(match[2]), 10, 64)
	if err != nil {
		return
	}
	firstTs, err := strconv.ParseInt(byteconv.UnsafeString(match[3]), 10, 64)
	if err != nil {
		return
	}
	lastTs, err := strconv.ParseInt(byteconv.UnsafeString(match[4]), 10, 64)
	if err != nil {
		return
	}
	return seekIndexFile{
		id:          id,
		first:       nano.Ts(firstTs),
		last:        nano.Ts(lastTs),
		recordCount: recordCount,
	}, true
}

// A tsDir is a directory found in the "<DataPath>/zd" directory of the archive,
// and holds all of the data & associated files for a span of time, currently
// fixed to a single day.
type tsDir struct {
	nano.Span
}

func tsDirFor(ts nano.Ts) tsDir {
	return tsDir{nano.Span{Ts: ts.Midnight(), Dur: nano.Day}}
}

func parseTsDirName(name string) (tsDir, bool) {
	t, err := time.Parse("20060102", name)
	if err != nil {
		return tsDir{}, false
	}
	return tsDirFor(nano.TimeToTs(t)), true
}

func (t tsDir) name() string {
	return t.Ts.Time().Format("20060102")
}

type tsDirVisitor func(tsd tsDir, si []SpanInfo) error

// tsDirVisit calls visitor for each tsDir in the archive, with all spans
// within the tsDir. tsDir's are visited in the archive's order, and the
// spans passed to visitor are also sorted by the archive's order.
func tsDirVisit(ctx context.Context, ark *Archive, filterSpan nano.Span, visitor tsDirVisitor) error {
	zdDir := ark.DataPath.AppendPath(dataDirname)
	dirents, err := iosrc.ReadDir(ctx, zdDir)
	if err != nil {
		return err
	}
	var tsdirs []tsDir
	for _, e := range dirents {
		if !e.IsDir() {
			continue
		}
		tsd, ok := parseTsDirName(e.Name())
		if !ok || !tsd.Overlaps(filterSpan) {
			continue
		}
		tsdirs = append(tsdirs, tsd)
	}
	sort.Slice(tsdirs, func(i, j int) bool {
		if ark.DataSortDirection == zbuf.DirTimeForward {
			return tsdirs[i].Ts < tsdirs[j].Ts
		}
		return tsdirs[j].Ts < tsdirs[i].Ts
	})
	for _, d := range tsdirs {
		dirents, err := iosrc.ReadDir(ctx, zdDir.AppendPath(d.name()))
		if err != nil {
			return err
		}
		if err := visitor(d, tsDirEntriesToSpans(ark, filterSpan, dirents)); err != nil {
			return err
		}
	}
	return nil
}

func tsDirEntriesToSpans(ark *Archive, filterSpan nano.Span, entries []iosrc.Info) []SpanInfo {
	dfileMap := make(map[ksuid.KSUID]struct{})
	sfileMap := make(map[ksuid.KSUID]seekIndexFile)
	for _, e := range entries {
		if df, ok := dataFileNameMatch(e.Name()); ok {
			dfileMap[df.id] = struct{}{}
			continue
		}
		if sf, ok := seekIndexNameMatch(e.Name()); ok {
			sfileMap[sf.id] = sf
			continue
		}
	}
	var si []SpanInfo
	for id, sf := range sfileMap {
		if !ark.filterAllowed(id) {
			continue
		}
		if !filterSpan.Overlaps(sf.span()) {
			continue
		}
		if _, ok := dfileMap[id]; !ok {
			continue
		}
		si = append(si, SpanInfo{
			First:       sf.first,
			Last:        sf.last,
			LogID:       newLogID(sf.first, id),
			RecordCount: sf.recordCount,
		})
	}
	sort.Slice(si, func(i, j int) bool {
		if ark.DataSortDirection == zbuf.DirTimeForward {
			if si[i].First == si[j].First {
				return si[i].LogID < si[j].LogID
			}
			return si[i].First < si[j].First
		} else {
			if si[j].First == si[i].First {
				return si[j].LogID < si[i].LogID
			}
			return si[j].First < si[i].First
		}
	})
	return si
}

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
func Walk(ctx context.Context, ark *Archive, visit Visitor) error {
	return SpanWalk(ctx, ark, func(_ SpanInfo, zardir iosrc.URI) error {
		return visit(zardir)
	})
}

// A LogID identifies a single zng file within an archive. It is created
// by doing a path join (with forward slashes, regardless of platform)
// of the relative location of the file under the archive's root directory.
type LogID string

func newLogID(ts nano.Ts, id ksuid.KSUID) LogID {
	return LogID(path.Join(dataDirname, tsDirFor(ts).name(), fmt.Sprintf("%s-%s.zng", fileKindData, id)))
}

// Path returns the local filesystem path for the log file, using the
// platforms file separator.
func (l LogID) Path(ark *Archive) iosrc.URI {
	return ark.DataPath.AppendPath(string(l))
}

type SpanInfo struct {
	First       nano.Ts // timestamp of first record in this span
	Last        nano.Ts // timestamp of last record in this span
	LogID       LogID
	RecordCount int64
}

// Span returns an inclusive nano.Span that contains both the first
// and last record timestamps.
func (si SpanInfo) Span() nano.Span {
	return nano.Span{Ts: si.First, Dur: 1}.Union(nano.Span{Ts: si.Last, Dur: 1})
}

func (si SpanInfo) Range(ark *Archive) string {
	return fmt.Sprintf("[%d-%d]", si.First, si.Last)
}

type SpanVisitor func(si SpanInfo, zardir iosrc.URI) error

func SpanWalk(ctx context.Context, ark *Archive, v SpanVisitor) error {
	return tsDirVisit(ctx, ark, nano.MaxSpan, func(_ tsDir, spans []SpanInfo) error {
		for _, s := range spans {
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
	})
}

// RmDirs descends a directory hierarchy looking for zar dirs and remove
// each such directory and all of its contents.
func RmDirs(ctx context.Context, ark *Archive) error {
	fn := func(u iosrc.URI) error {
		return ark.dataSrc.RemoveAll(ctx, u)
	}
	return Walk(ctx, ark, fn)
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
	return SpanWalk(ctx, ams.ark, func(si SpanInfo, zardir iosrc.URI) error {
		if !sf.Span.Overlaps(si.Span()) {
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
				paths = append(paths, Localize(zardir, input).String())
			}
			rc := detector.MultiFileReader(zctx, paths, zio.ReaderOpts{Format: "zng"})
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
