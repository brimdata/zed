package archive

import (
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqe"
)

const metadataFilename = "zar.json"

type Metadata struct {
	Version           int                  `json:"version"`
	DataPath          string               `json:"data_path"`
	LogSizeThreshold  int64                `json:"log_size_threshold"`
	DataSortDirection zbuf.Direction       `json:"data_sort_direction"`
	Spans             []SpanInfo           `json:"spans"`
	Indexes           map[string]IndexInfo `json:"indexes"`
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

type SpanInfo struct {
	Span  nano.Span `json:"span"`
	LogID LogID     `json:"log_id"`
}

type IndexInfo struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

func (c *Metadata) Write(uri iosrc.URI) error {
	switch uri.Scheme {
	case "file":
		path := uri.Filepath()
		return fs.MarshalJSONFile(c, path, 0600)
	case "s3":
		return zqe.E("s3 not handled yet: %s", uri)
	default:
		return zqe.E("unhandled uriu scheme: %s", uri)
	}
}

func MetadataRead(uri iosrc.URI) (*Metadata, time.Time, error) {
	// Read the mtime before the read so that the returned time
	// represents a time at or before the content of the metadata file.
	switch uri.Scheme {
	case "file":
		path := uri.Filepath()
		fi, err := os.Stat(path)
		if err != nil {
			return nil, time.Time{}, err
		}
		var c Metadata
		if err := fs.UnmarshalJSONFile(path, &c); err != nil {
			return nil, time.Time{}, err
		}
		return &c, fi.ModTime(), nil
	case "s3":
		return nil, time.Time{}, zqe.E("s3 not handled yet: %s", uri)
	default:
		return nil, time.Time{}, zqe.E("unhandled uri scheme: %s", uri)
	}
}

const (
	DefaultLogSizeThreshold  = 500 * 1024 * 1024
	DefaultDataSortDirection = zbuf.DirTimeReverse
)

type CreateOptions struct {
	LogSizeThreshold *int64
	DataPath         string
}

func (c *CreateOptions) toMetadata() *Metadata {
	m := &Metadata{
		Version:           0,
		LogSizeThreshold:  DefaultLogSizeThreshold,
		DataSortDirection: DefaultDataSortDirection,
		DataPath:          ".",
		Indexes:           make(map[string]IndexInfo),
	}

	if c.LogSizeThreshold != nil {
		m.LogSizeThreshold = *c.LogSizeThreshold
	}
	if c.DataPath != "" {
		m.DataPath = c.DataPath
	}

	return m
}

type Archive struct {
	Root              iosrc.URI
	DataPath          iosrc.URI
	DataSortDirection zbuf.Direction
	LogSizeThreshold  int64
	LogsFiltered      bool

	dataSrc iosrc.Source

	// mu protects below fields.
	mu      sync.RWMutex
	indexes map[string]IndexInfo // map key is index path
	spans   []SpanInfo
	// mdModTime is the mtime of the metadata at or before its contents
	// were last read.
	mdModTime time.Time
	// mdUpdateCount is incremented every time the metadata for the
	// archive is potentially updated due to writing new logs, or
	// on re-reading the metadata file.
	mdUpdateCount int
}

func (ark *Archive) AppendSpans(spans []SpanInfo) error {
	if ark.LogsFiltered {
		return errors.New("cannot add spans to log filtered archive")
	}

	ark.mu.Lock()
	defer ark.mu.Unlock()

	ark.spans = append(ark.spans, spans...)

	sort.Slice(ark.spans, func(i, j int) bool {
		if ark.DataSortDirection == zbuf.DirTimeForward {
			return ark.spans[i].Span.Ts < ark.spans[j].Span.Ts
		}
		return ark.spans[j].Span.Ts < ark.spans[i].Span.Ts
	})

	err := ark.metaWrite()
	if err != nil {
		return err
	}

	ark.mdUpdateCount++
	return nil
}

func (ark *Archive) AddIndexes(indexes []IndexInfo) error {
	ark.mu.Lock()
	defer ark.mu.Unlock()
	for _, ind := range indexes {
		ark.indexes[ind.Path] = ind
	}
	err := ark.metaWrite()
	if err != nil {
		return err
	}

	ark.mdUpdateCount++
	return nil
}

func (ark *Archive) metaWrite() error {
	m := &Metadata{
		Version:           0,
		LogSizeThreshold:  ark.LogSizeThreshold,
		DataSortDirection: ark.DataSortDirection,
		DataPath:          ark.DataPath.String(),
		Indexes:           ark.indexes,
		Spans:             ark.spans,
	}
	return m.Write(ark.mdURI())
}

func (ark *Archive) mdURI() iosrc.URI {
	return ark.Root.AppendPath(metadataFilename)
}

// UpdateCheck looks at the archive's metadata file to see if it
// has been written to since last read; if so, it is read and the
// available spans are updated. A counter is returned, starting
// from 1, which is incremented every time this Archive has re-read
// the metadata file, and possibly updated the available spans.
func (ark *Archive) UpdateCheck() (int, error) {
	if ark.LogsFiltered {
		// If a logfilter was specified at open, there's no need to
		// check for newer log files in the archive.
		return ark.mdUpdateCount, nil
	}

	fi, err := iosrc.Stat(ark.mdURI())
	if err != nil {
		return 0, err
	}

	ark.mu.RLock()
	if fi.ModTime().Equal(ark.mdModTime) {
		cnt := ark.mdUpdateCount
		ark.mu.RUnlock()
		return cnt, nil
	}
	ark.mu.RUnlock()

	ark.mu.Lock()
	defer ark.mu.Unlock()

	if fi.ModTime().Equal(ark.mdModTime) {
		return ark.mdUpdateCount, nil
	}

	md, mtime, err := MetadataRead(ark.mdURI())
	if err != nil {
		return 0, err
	}

	ark.spans = md.Spans
	ark.mdModTime = mtime
	ark.mdUpdateCount++
	return ark.mdUpdateCount, nil
}

type OpenOptions struct {
	LogFilter  []string
	DataSource iosrc.Source
}

func OpenArchive(rpath string, oo *OpenOptions) (*Archive, error) {
	root, err := iosrc.ParseURI(rpath)
	if err != nil {
		return nil, err
	}
	m, mtime, err := MetadataRead(root.AppendPath(metadataFilename))
	if err != nil {
		return nil, err
	}
	if m.DataPath == "." {
		m.DataPath = rpath
	}
	dpuri, err := iosrc.ParseURI(m.DataPath)
	if err != nil {
		return nil, err
	}

	ark := &Archive{
		Root:              root,
		DataSortDirection: m.DataSortDirection,
		LogSizeThreshold:  m.LogSizeThreshold,
		DataPath:          dpuri,
		indexes:           m.Indexes,
		mdModTime:         mtime,
		mdUpdateCount:     1,
	}

	if oo != nil && oo.DataSource != nil {
		ark.dataSrc = oo.DataSource
	} else {
		if ark.dataSrc, err = iosrc.GetSource(dpuri); err != nil {
			return nil, err
		}
	}
	if oo != nil && len(oo.LogFilter) != 0 {
		ark.LogsFiltered = true
		lmap := make(map[LogID]struct{})
		for _, l := range oo.LogFilter {
			lmap[LogID(l)] = struct{}{}
		}

		for _, s := range m.Spans {
			if _, ok := lmap[s.LogID]; ok {
				ark.spans = append(ark.spans, s)
			}
		}
		if len(ark.spans) == 0 {
			return nil, zqe.E(zqe.Invalid, "OpenArchive: no logs left after filter")
		}
	} else {
		ark.spans = m.Spans
	}

	return ark, nil
}

func CreateOrOpenArchive(rpath string, co *CreateOptions, oo *OpenOptions) (*Archive, error) {
	root, err := iosrc.ParseURI(rpath)
	if err != nil {
		return nil, err
	}

	mdPath := root.AppendPath(metadataFilename)
	ok, err := iosrc.Exists(mdPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		src, err := iosrc.GetSource(root)
		if err != nil {
			return nil, err
		}
		if dm, ok := src.(iosrc.DirMaker); ok {
			if err := dm.MkdirAll(root, 0700); err != nil {
				return nil, err
			}
		}
		if err := co.toMetadata().Write(mdPath); err != nil {
			return nil, err
		}
	}

	return OpenArchive(rpath, oo)
}
