package archive

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

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

func writeTempFile(dir, pattern string, b []byte) (name string, err error) {
	f, err := ioutil.TempFile(dir, pattern)
	if err != nil {
		return "", err
	}
	_, err = f.Write(b)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	err = f.Close()
	if err != nil {
		os.Remove(f.Name())
	}
	return f.Name(), nil
}

func (c *Metadata) Write(path string) (err error) {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := writeTempFile(filepath.Dir(path), "."+metadataFilename+".*", b)
	if err != nil {
		return err
	}
	err = os.Rename(tmp, path)
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

func ConfigRead(path string) (*Metadata, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var m Metadata
	return &m, json.NewDecoder(f).Decode(&m)
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
	Meta *Metadata
	Root string

	// Spans contains either all spans from metadata, or a subset
	// due to opening the archive with a filter list.
	Spans []SpanInfo
}

func (ark *Archive) AppendSpans(spans []SpanInfo) error {
	ark.Meta.Spans = append(ark.Meta.Spans, spans...)

	sort.Slice(ark.Meta.Spans, func(i, j int) bool {
		if ark.Meta.DataSortDirection == zbuf.DirTimeForward {
			return ark.Meta.Spans[i].Span.Ts < ark.Meta.Spans[j].Span.Ts
		}
		return ark.Meta.Spans[j].Span.Ts < ark.Meta.Spans[i].Span.Ts
	})

	return ark.Meta.Write(filepath.Join(ark.Root, metadataFilename))
}

type OpenOptions struct {
	LogFilter []string
}

func OpenArchive(path string, oo *OpenOptions) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no archive directory specified")
	}
	c, err := ConfigRead(filepath.Join(path, metadataFilename))
	if err != nil {
		return nil, err
	}

	var spans []SpanInfo
	if oo != nil && len(oo.LogFilter) != 0 {
		lmap := make(map[LogID]struct{})
		for _, l := range oo.LogFilter {
			lmap[LogID(l)] = struct{}{}
		}
		for _, s := range c.Spans {
			if _, ok := lmap[s.LogID]; ok {
				spans = append(spans, s)
			}
		}
		if len(spans) == 0 {
			return nil, zqe.E(zqe.Invalid, "OpenArchive: no spans left after filter")
		}
	} else {
		spans = c.Spans
	}

	return &Archive{
		Meta:  c,
		Root:  path,
		Spans: spans,
	}, nil
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
