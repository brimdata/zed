package archive

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
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
	FileKindUnknown       FileKind = ""
	FileKindData                   = "d"
	FileKindDataCompacted          = "dc"
	FileKindMetadata               = "m"
	FileKindSeek                   = "ts"
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
		if ark.DataOrder == zbuf.OrderAsc {
			return tsdirs[i].Ts < tsdirs[j].Ts
		}
		return tsdirs[j].Ts < tsdirs[i].Ts
	})
	for _, d := range tsdirs {
		dirents, err := iosrc.ReadDir(ctx, zdDir.AppendPath(d.name()))
		if err != nil {
			return err
		}
		chunks, err := tsDirEntriesToChunks(ctx, ark, filterSpan, d, dirents)
		if err != nil {
			return err
		}
		if err := visitor(d, chunks); err != nil {
			return err
		}
	}
	return nil
}

func tsDirEntriesToChunks(ctx context.Context, ark *Archive, filterSpan nano.Span, tsDir tsDir, entries []iosrc.Info) ([]Chunk, error) {
	type seen struct {
		data FileKind
		meta bool
	}
	m := make(map[ksuid.KSUID]seen)
	for _, e := range entries {
		if kind, id, ok := chunkFileMatch(e.Name()); ok {
			if !ark.filterAllowed(id) {
				continue
			}
			s := m[id]
			switch kind {
			case FileKindData, FileKindDataCompacted:
				s.data = kind
			case FileKindMetadata:
				s.meta = true
			}
			m[id] = s
		}
	}
	var chunks []Chunk
	for id, seen := range m {
		if !seen.meta || (seen.data == FileKindUnknown) {
			continue
		}
		md, err := readChunkMetadata(ctx, chunkMetadataPath(ark, tsDir, id))
		if err != nil {
			if zqe.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		chunk := md.Chunk(id)
		if !filterSpan.Overlaps(chunk.Span()) {
			continue
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

var chunkFileRegex = regexp.MustCompile(`(d|dc|m)-([0-9A-Za-z]{27}).zng$`)

func chunkFileMatch(s string) (kind FileKind, id ksuid.KSUID, ok bool) {
	match := chunkFileRegex.FindStringSubmatch(s)
	if match == nil {
		return
	}
	k := FileKind(match[1])
	switch k {
	case FileKindData:
	case FileKindDataCompacted:
	case FileKindMetadata:
	default:
		return
	}
	id, err := ksuid.Parse(match[2])
	if err != nil {
		return
	}
	return k, id, true
}

// A Chunk is a file that holds records ordered according to the archive's
// data order.
// seekIndexPath returns the path of an associated microindex written at import
// time, which can be used to lookup a nearby seek offset for a desired
// timestamp.
// metadataPath returns the path of an associated zng file that holds
// information about the records in the chunk, including the total number,
// and the first and last record timestamps.
type Chunk struct {
	Id          ksuid.KSUID
	First       nano.Ts
	Last        nano.Ts
	Kind        FileKind
	RecordCount uint64
	Masks       []ksuid.KSUID
}

func (c Chunk) tsDir() tsDir {
	return newTsDir(c.First)
}

func (c Chunk) seekIndexPath(ark *Archive) iosrc.URI {
	return c.tsDir().path(ark).AppendPath(fmt.Sprintf("%s-%s.zng", FileKindSeek, c.Id))
}

func (c Chunk) metadataPath(ark *Archive) iosrc.URI {
	return chunkMetadataPath(ark, c.tsDir(), c.Id)
}

func (c Chunk) Span() nano.Span {
	return firstLastToSpan(c.First, c.Last)
}

func (c Chunk) RelativePath() string {
	return chunkRelativePath(c.tsDir(), c.Kind, c.Id)
}

func chunkRelativePath(tsd tsDir, kind FileKind, id ksuid.KSUID) string {
	return path.Join(dataDirname, tsd.name(), fmt.Sprintf("%s-%s.zng", kind, id))
}

func parseChunkRelativePath(s string) (tsDir, FileKind, ksuid.KSUID, bool) {
	ss := strings.Split(s, "/")
	if len(ss) < 2 {
		return tsDir{}, FileKindUnknown, ksuid.Nil, false
	}
	kind, id, ok := chunkFileMatch(ss[len(ss)-1])
	if !ok {
		return tsDir{}, FileKindUnknown, ksuid.Nil, false
	}
	tsd, ok := parseTsDirName(ss[len(ss)-2])
	if !ok {
		return tsDir{}, FileKindUnknown, ksuid.Nil, false
	}
	return tsd, kind, id, true
}

// ZarDir returns a URI for a directory specific to this data file, expected
// to hold microindexes or other files associated with this chunk's data.
func (c Chunk) ZarDir(ark *Archive) iosrc.URI {
	return ark.DataPath.AppendPath(string(c.RelativePath() + zarExt))
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
	return chunkPath(ark, c.tsDir(), c.Kind, c.Id)
}

func chunkPath(ark *Archive, tsd tsDir, kind FileKind, id ksuid.KSUID) iosrc.URI {
	return ark.DataPath.AppendPath(chunkRelativePath(tsd, kind, id))
}

func (c Chunk) Range() string {
	return fmt.Sprintf("[%d-%d]", c.First, c.Last)
}

// Remove deletes the data, metadata, seek, and any other associated files
// with the chunk, including the zar directory. Any 'not found' errors will
// be ignored.
func (c Chunk) Remove(ctx context.Context, ark *Archive) error {
	uris := []iosrc.URI{
		c.Path(ark),
		c.ZarDir(ark),
		c.metadataPath(ark),
		c.seekIndexPath(ark),
	}
	for _, u := range uris {
		if err := ark.dataSrc.RemoveAll(ctx, u); err != nil && !zqe.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func chunkLess(order zbuf.Order, a, b Chunk) bool {
	if order == zbuf.OrderDesc {
		a, b = b, a
	}
	switch {
	case a.First != b.First:
		return a.First < b.First
	case a.Last != b.Last:
		return a.Last < b.Last
	case a.RecordCount != b.RecordCount:
		return a.RecordCount < b.RecordCount
	}
	return ksuid.Compare(a.Id, b.Id) < 0
}

func chunksSort(order zbuf.Order, c []Chunk) {
	sort.Slice(c, func(i, j int) bool {
		return chunkLess(order, c[i], c[j])
	})
}

type chunkMetadata struct {
	Kind        FileKind
	First       nano.Ts
	Last        nano.Ts
	RecordCount uint64
	Masks       []ksuid.KSUID
}

func (md chunkMetadata) Chunk(id ksuid.KSUID) Chunk {
	return Chunk{
		Id:          id,
		First:       md.First,
		Last:        md.Last,
		Kind:        md.Kind,
		RecordCount: md.RecordCount,
		Masks:       md.Masks,
	}
}

func chunkMetadataPath(ark *Archive, tsDir tsDir, id ksuid.KSUID) iosrc.URI {
	return tsDir.path(ark).AppendPath(fmt.Sprintf("%s-%s.zng", FileKindMetadata, id))
}

func writeChunkMetadata(ctx context.Context, uri iosrc.URI, md chunkMetadata) error {
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, md)
	if err != nil {
		return err
	}
	out, err := iosrc.NewWriter(ctx, uri)
	if err != nil {
		return err
	}
	zw := zngio.NewWriter(bufwriter.New(out), zngio.WriterOpts{})
	if err := zw.Write(rec); err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}

func readChunkMetadata(ctx context.Context, uri iosrc.URI) (chunkMetadata, error) {
	in, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return chunkMetadata{}, err
	}
	defer in.Close()
	zctx := resolver.NewContext()
	zr := zngio.NewReader(in, zctx)
	rec, err := zr.Read()
	if err != nil {
		return chunkMetadata{}, err
	}
	var md chunkMetadata
	if err := resolver.UnmarshalRecord(zctx, rec, &md); err != nil {
		return chunkMetadata{}, err
	}
	return md, nil
}

type Visitor func(chunk Chunk) error

// Walk calls visitor for every data chunk in the archive.
func Walk(ctx context.Context, ark *Archive, v Visitor) error {
	dirmkr, _ := ark.dataSrc.(iosrc.DirMaker)
	return tsDirVisit(ctx, ark, nano.MaxSpan, func(_ tsDir, chunks []Chunk) error {
		chunksSort(ark.DataOrder, chunks)
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

func (s SpanInfo) ChunkRange(order zbuf.Order, chunkIdx int) string {
	first, last := spanToFirstLast(order, s.Span)
	c := s.Chunks[chunkIdx]
	return fmt.Sprintf("[%d-%d,%d-%d]", first, last, c.First, c.Last)
}

func (si *SpanInfo) RemoveMasked() []Chunk {
	if len(si.Chunks) == 1 {
		return nil
	}
	var maskIds []ksuid.KSUID
	for _, c := range si.Chunks {
		for _, mid := range c.Masks {
			maskIds = append(maskIds, mid)
		}
	}
	if len(maskIds) == 0 {
		return nil
	}
	var chunks []Chunk
	var removed []Chunk
	for _, c := range si.Chunks {
		var masked bool
		for _, mid := range maskIds {
			if mid == c.Id {
				masked = true
				removed = append(removed, c)
				break
			}
		}
		if !masked {
			chunks = append(chunks, c)
		}
	}
	// No need to sort chunks since we perform the mask removal in-order.
	si.Chunks = chunks
	return removed
}

// SpanWalk calls visitor with each SpanInfo within the filter span.
func SpanWalk(ctx context.Context, ark *Archive, filter nano.Span, visitor func(si SpanInfo) error) error {
	dirmkr, _ := ark.dataSrc.(iosrc.DirMaker)
	return tsDirVisit(ctx, ark, filter, func(_ tsDir, chunks []Chunk) error {
		sinfos := mergeChunksToSpans(chunks, ark.DataOrder, filter)
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
