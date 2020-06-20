package archive

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqe"
)

const metadataFilename = "zar.json"

type Metadata struct {
	Version           int            `json:"version"`
	LogSizeThreshold  int64          `json:"log_size_threshold"`
	DataSortDirection zbuf.Direction `json:"data_sort_direction"`
	Spans             []SpanInfo     `json:"spans"`
}

// A LogID identifies a single zng file within an archive. It is created
// by doing a path join (with forward slashes, regardless of platform)
// of the relative location of the file under the archive's root directory.
type LogID string

// Path returns the local filesystem path for the log file, using the
// platforms file separator.
func (l LogID) Path(ark *Archive) string {
	return filepath.Join(ark.Root, filepath.FromSlash(string(l)))
}

type SpanInfo struct {
	Span  nano.Span `json:"span"`
	LogID LogID     `json:"log_id"`
}

func (c *Metadata) Write(path string) error {
	return fs.MarshalJSONFile(c, path, 0600)
}

func MetadataRead(path string) (*Metadata, time.Time, error) {
	// Read the mtime before the read so that the returned time
	// represents a time at or before the content of the metadata file.
	fi, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	var c Metadata
	if err := fs.UnmarshalJSONFile(path, &c); err != nil {
		return nil, time.Time{}, err
	}
	return &c, fi.ModTime(), nil
}

const (
	DefaultLogSizeThreshold  = 500 * 1024 * 1024
	DefaultDataSortDirection = zbuf.DirTimeReverse
)

type CreateOptions struct {
	LogSizeThreshold *int64
}

func (c *CreateOptions) toMetadata() *Metadata {
	m := &Metadata{
		Version:           0,
		LogSizeThreshold:  DefaultLogSizeThreshold,
		DataSortDirection: DefaultDataSortDirection,
	}

	if c.LogSizeThreshold != nil {
		m.LogSizeThreshold = *c.LogSizeThreshold
	}

	return m
}

type Archive struct {
	Root              string
	DataSortDirection zbuf.Direction
	LogSizeThreshold  int64
	LogsFiltered      bool

	// mu protects below fields.
	mu    sync.RWMutex
	spans []SpanInfo
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

func (ark *Archive) metaWrite() error {
	m := &Metadata{
		Version:           0,
		LogSizeThreshold:  ark.LogSizeThreshold,
		DataSortDirection: ark.DataSortDirection,
		Spans:             ark.spans,
	}
	return m.Write(ark.mdPath())
}

func (ark *Archive) mdPath() string {
	return filepath.Join(ark.Root, metadataFilename)
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

	fi, err := os.Stat(ark.mdPath())
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

	md, mtime, err := MetadataRead(ark.mdPath())
	if err != nil {
		return 0, err
	}

	ark.spans = md.Spans
	ark.mdModTime = mtime
	ark.mdUpdateCount++
	return ark.mdUpdateCount, nil
}

type OpenOptions struct {
	LogFilter []string
}

func OpenArchive(path string, oo *OpenOptions) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no archive directory specified")
	}
	m, mtime, err := MetadataRead(filepath.Join(path, metadataFilename))
	if err != nil {
		return nil, err
	}

	ark := &Archive{
		Root:              path,
		DataSortDirection: m.DataSortDirection,
		LogSizeThreshold:  m.LogSizeThreshold,
		mdModTime:         mtime,
		mdUpdateCount:     1,
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

func CreateOrOpenArchive(path string, co *CreateOptions, oo *OpenOptions) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no archive directory specified")
	}
	cfgpath := filepath.Join(path, metadataFilename)
	if _, err := os.Stat(cfgpath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0700); err != nil {
				return nil, err
			}
			err = co.toMetadata().Write(cfgpath)
		}
		if err != nil {
			return nil, err
		}
	}
	return OpenArchive(path, oo)
}
