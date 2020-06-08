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
)

const metaDataFilename = "zar.json"

var DefaultConfig Metadata = Metadata{
	Version:           0,
	LogSizeThreshold:  500 * 1024 * 1024,
	DataSortDirection: zbuf.DirTimeReverse,
}

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
	tmp, err := writeTempFile(filepath.Dir(path), "."+metaDataFilename+".*", b)
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

type CreateOptions struct {
	LogSizeThreshold *int64
}

func (c *CreateOptions) toMetadata() *Metadata {
	cfg := DefaultConfig

	if c.LogSizeThreshold != nil {
		cfg.LogSizeThreshold = *c.LogSizeThreshold
	}

	return &cfg
}

type Archive struct {
	Meta *Metadata
	Root string
}

func (ark *Archive) AppendSpans(spans []SpanInfo) error {
	ark.Meta.Spans = append(ark.Meta.Spans, spans...)

	sort.Slice(ark.Meta.Spans, func(i, j int) bool {
		if ark.Meta.DataSortDirection == zbuf.DirTimeForward {
			return ark.Meta.Spans[i].Span.Ts < ark.Meta.Spans[j].Span.Ts
		}
		return ark.Meta.Spans[j].Span.Ts < ark.Meta.Spans[i].Span.Ts
	})

	return ark.Meta.Write(filepath.Join(ark.Root, metaDataFilename))
}

func OpenArchive(path string) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no archive directory specified")
	}
	c, err := ConfigRead(filepath.Join(path, metaDataFilename))
	if err != nil {
		return nil, err
	}

	return &Archive{
		Meta: c,
		Root: path,
	}, nil
}

func CreateOrOpenArchive(path string, co *CreateOptions) (*Archive, error) {
	if path == "" {
		return nil, errors.New("no archive directory specified")
	}
	cfgpath := filepath.Join(path, metaDataFilename)
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
	return OpenArchive(path)
}
