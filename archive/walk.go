package archive

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/segmentio/ksuid"
)

const (
	dataDirname = "zd"
	zarExt      = ".zar"
)

// A FileKind is the first part of a file name, used to differentiate files
// when they are listed from the archive's backing store.
type FileKind string

const (
	FileKindUnknown FileKind = ""
	FileKindData             = "d"
	FileKindSeek             = "ts"
)

// A tsDir is a directory found in the "<DataPath>/zd" directory of the archive,
// and holds all of the data & associated files for a span of time, currently
// fixed to a single day.
type tsDir struct {
	nano.Span
}

func newTsDir(ts nano.Ts) tsDir {
	return tsDir{nano.Span{Ts: ts.Midnight(), Dur: nano.Day}}
}

func parseTsDirName(name string) (tsDir, bool) {
	t, err := time.Parse("20060102", name)
	if err != nil {
		return tsDir{}, false
	}
	return newTsDir(nano.TimeToTs(t)), true
}

func (t tsDir) path(ark *Archive) iosrc.URI {
	return ark.DataPath.AppendPath(dataDirname, t.name())
}

func (t tsDir) name() string {
	return t.Ts.Time().Format("20060102")
}

type tsDirVisitor func(tsd tsDir, unsortedChunks []Chunk) error

// tsDirVisit calls visitor for each tsDir whose span overlaps with the
// given span. tsDirs are visited in the archive's order, but the
// chunks passed to visitor are not sorted.
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
		if err := visitor(d, tsDirEntriesToChunks(ark, filterSpan, dirents)); err != nil {
			return err
		}
	}
	return nil
}

func tsDirEntriesToChunks(ark *Archive, filterSpan nano.Span, entries []iosrc.Info) []Chunk {
	var chunks []Chunk
	for _, e := range entries {
		chunk, ok := ChunkNameMatch(e.Name())
		if !ok {
			continue
		}
		if !ark.filterAllowed(chunk.Id) {
			continue
		}
		if !filterSpan.Overlaps(chunk.Span()) {
			continue
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}

// A LogID identifies a single zng file within an archive. It is created
// by doing a path join (with forward slashes, regardless of platform)
// of the relative location of the file under the archive's root directory.
type LogID string

// Path returns the local filesystem path for the log file, using the
// platforms file separator.
func (l LogID) Path(ark *Archive) iosrc.URI {
	return ark.DataPath.AppendPath(string(l))
}

// A Chunk is a file that holds records ordered according to the archive's
// data order. The name of the file encodes the number of records it contains
// and the timestamps of its first & last records. seekIndexPath returns the
// path of an associated microindex written at import time, which can be used
// to lookup a nearby seek offset for a desired timestamp.
type Chunk struct {
	Id          ksuid.KSUID
	First       nano.Ts
	Last        nano.Ts
	Kind        FileKind
	RecordCount int64
}

var chunkNameRegex = regexp.MustCompile(`d-([0-9A-Za-z]{27})-([0-9]+)-([0-9]+)-([0-9]+).zng$`)

func ChunkNameMatch(s string) (c Chunk, ok bool) {
	match := chunkNameRegex.FindStringSubmatch(s)
	if match == nil {
		return
	}
	id, err := ksuid.Parse(match[1])
	if err != nil {
		return
	}
	recordCount, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return
	}
	firstTs, err := strconv.ParseInt(match[3], 10, 64)
	if err != nil {
		return
	}
	lastTs, err := strconv.ParseInt(match[4], 10, 64)
	if err != nil {
		return
	}
	return Chunk{
		Id:          id,
		First:       nano.Ts(firstTs),
		Last:        nano.Ts(lastTs),
		Kind:        FileKindData,
		RecordCount: recordCount,
	}, true
}

func (c Chunk) tsDir() tsDir {
	return newTsDir(c.First)
}

func (c Chunk) seekIndexPath(ark *Archive) iosrc.URI {
	return c.tsDir().path(ark).AppendPath(fmt.Sprintf("%s-%s.zng", FileKindSeek, c.Id))
}

func (c Chunk) Span() nano.Span {
	return firstLastToSpan(c.First, c.Last)
}

func (c Chunk) LogID() LogID {
	name := fmt.Sprintf("%s-%s-%d-%d-%d.zng", c.Kind, c.Id, c.RecordCount, c.First, c.Last)
	return LogID(path.Join(dataDirname, newTsDir(c.First).name(), name))
}

// ZarDir returns a URI for a directory specific to this data file, expected
// to hold microindexes or other files associated with this chunk's data.
func (c Chunk) ZarDir(ark *Archive) iosrc.URI {
	return ark.DataPath.AppendPath(string(c.LogID() + zarExt))
}

// Localize returns a URI that joins the provided relative path name to the
// zardir for this chunk. The special name "_" is mapped to the path of the
// data file for this chunk.
func (c Chunk) Localize(ark *Archive, pathname string) iosrc.URI {
	if pathname == "_" {
		return c.Path(ark)
	}
	return c.ZarDir(ark).AppendPath(pathname)
}

func (c Chunk) Path(ark *Archive) iosrc.URI {
	return ark.DataPath.AppendPath(string(c.LogID()))
}

func (c Chunk) Range() string {
	return fmt.Sprintf("[%d-%d]", c.First, c.Last)
}

func chunkTsLess(dir zbuf.Direction, iTs nano.Ts, iKid ksuid.KSUID, jTs nano.Ts, jKid ksuid.KSUID) bool {
	if dir == zbuf.DirTimeForward {
		if iTs == jTs {
			return ksuid.Compare(iKid, jKid) < 0
		}
		return iTs < jTs
	}
	if jTs == iTs {
		return ksuid.Compare(jKid, iKid) < 0
	}
	return jTs < iTs
}

func chunksSort(dir zbuf.Direction, c []Chunk) {
	sort.Slice(c, func(i, j int) bool {
		return chunkTsLess(dir, c[i].First, c[i].Id, c[j].First, c[j].Id)
	})
}

type Visitor func(chunk Chunk) error

// Walk calls visitor for every data chunk in the archive.
func Walk(ctx context.Context, ark *Archive, v Visitor) error {
	dirmkr, _ := ark.dataSrc.(iosrc.DirMaker)
	return tsDirVisit(ctx, ark, nano.MaxSpan, func(_ tsDir, chunks []Chunk) error {
		chunksSort(ark.DataSortDirection, chunks)
		for _, c := range chunks {
			if dirmkr != nil {
				zardir := c.ZarDir(ark)
				if err := dirmkr.MkdirAll(zardir, 0700); err != nil {
					return err
				}
			}
			if err := v(c); err != nil {
				return err
			}
		}
		return nil
	})
}

// RmDirs descends a directory hierarchy looking for zar dirs and remove
// each such directory and all of its contents.
func RmDirs(ctx context.Context, ark *Archive) error {
	return Walk(ctx, ark, func(chunk Chunk) error {
		return ark.dataSrc.RemoveAll(ctx, chunk.ZarDir(ark))
	})
}

// A SpanInfo is a logical view of the records within a time span, stored
// in one or more Chunks.
type SpanInfo struct {
	Span nano.Span

	// Chunks are the data files that contain records within this SpanInfo.
	// The Chunks may have spans that extend beyond this SpanInfo, so any
	// records from these Chunks should be limited to those that fall within
	// this SpanInfo's Span.
	Chunks []Chunk
}

func (s SpanInfo) ChunkRange(dir zbuf.Direction, chunkIdx int) string {
	first, last := spanToFirstLast(dir, s.Span)
	c := s.Chunks[chunkIdx]
	return fmt.Sprintf("[%d-%d,%d-%d]", first, last, c.First, c.Last)
}

// SpanWalk calls visitor with each SpanInfo within the filter span.
func SpanWalk(ctx context.Context, ark *Archive, filter nano.Span, visitor func(si SpanInfo) error) error {
	dirmkr, _ := ark.dataSrc.(iosrc.DirMaker)
	return tsDirVisit(ctx, ark, filter, func(_ tsDir, chunks []Chunk) error {
		sinfos := mergeChunksToSpans(chunks, ark.DataSortDirection, filter)
		for _, s := range sinfos {
			if dirmkr != nil {
				for _, c := range s.Chunks {
					zardir := c.ZarDir(ark)
					if err := dirmkr.MkdirAll(zardir, 0700); err != nil {
						return err
					}
				}
			}
			if err := visitor(s); err != nil {
				return err
			}
		}
		return nil
	})
}
