package lake

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/lake/immcache"
	"github.com/brimsec/zq/ppl/lake/index"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqe"
	"github.com/segmentio/ksuid"
)

const (
	metadataFilename = "zar.json"
	indexdefsDir     = "indexdefs"
)

type Metadata struct {
	Version int `json:"version"`

	DataPath         string     `json:"data_path"`
	DataOrder        zbuf.Order `json:"data_order"`
	LogSizeThreshold int64      `json:"log_size_threshold"`
}

func (c *Metadata) Write(uri iosrc.URI) error {
	err := iosrc.Replace(context.Background(), uri, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(c)
	})
	if err != nil {
		return err
	}
	if uri.Scheme == "file" {
		// Ensure the mtime is updated on the file after the close. This Chtimes
		// call was required due to failures seen in CI, when an mtime change
		// wasn't observed after some writes.
		// See https://github.com/brimsec/brim/issues/883.
		now := time.Now()
		return os.Chtimes(uri.Filepath(), now, now)
	}
	return nil
}

func MetadataRead(ctx context.Context, uri iosrc.URI) (*Metadata, error) {
	rc, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var md Metadata
	if err := json.NewDecoder(rc).Decode(&md); err != nil {
		return nil, err
	}
	return &md, nil
}

const (
	DefaultDataOrder        = zbuf.OrderDesc
	DefaultLogSizeThreshold = 500 * 1024 * 1024
)

type CreateOptions struct {
	DataPath         string
	LogSizeThreshold *int64
	SortAscending    bool
}

func (c *CreateOptions) toMetadata() *Metadata {
	m := &Metadata{
		Version:          0,
		LogSizeThreshold: DefaultLogSizeThreshold,
		DataOrder:        DefaultDataOrder,
		DataPath:         ".",
	}
	if c.LogSizeThreshold != nil {
		m.LogSizeThreshold = *c.LogSizeThreshold
	}
	if c.DataPath != "" {
		m.DataPath = c.DataPath
	}
	if c.SortAscending {
		m.DataOrder = zbuf.OrderAsc
	}
	return m
}

type Lake struct {
	Root             iosrc.URI
	DataPath         iosrc.URI
	DataOrder        zbuf.Order
	LogSizeThreshold int64
	LogFilter        []ksuid.KSUID

	immfiles interface {
		ReadFile(context.Context, iosrc.URI) ([]byte, error)
	}
}

func (lk *Lake) metaWrite() error {
	m := &Metadata{
		Version:          0,
		LogSizeThreshold: lk.LogSizeThreshold,
		DataOrder:        lk.DataOrder,
		DataPath:         lk.DataPath.String(),
	}
	return m.Write(lk.mdURI())
}

func (lk *Lake) mdURI() iosrc.URI {
	return lk.Root.AppendPath(metadataFilename)
}

func (lk *Lake) filterAllowed(id ksuid.KSUID) bool {
	if len(lk.LogFilter) == 0 {
		return true
	}
	for _, fid := range lk.LogFilter {
		if fid == id {
			return true
		}
	}
	return false
}

func (lk *Lake) DefinitionsDir() iosrc.URI {
	return lk.Root.AppendPath(indexdefsDir)
}

func (lk *Lake) ReadDefinitions(ctx context.Context) (index.Definitions, error) {
	defs, err := index.ReadDefinitions(ctx, lk.DefinitionsDir())
	if zqe.IsNotFound(err) {
		err = iosrc.MkdirAll(lk.DefinitionsDir(), 0700)
	}
	return defs, err
}

type OpenOptions struct {
	ImmutableCache immcache.ImmutableCache
}

func OpenLake(rpath string, oo *OpenOptions) (*Lake, error) {
	return OpenLakeWithContext(context.Background(), rpath, oo)
}

func OpenLakeWithContext(ctx context.Context, rpath string, oo *OpenOptions) (*Lake, error) {
	root, err := iosrc.ParseURI(rpath)
	if err != nil {
		return nil, err
	}
	return openLake(ctx, root, oo)
}

func openLake(ctx context.Context, root iosrc.URI, oo *OpenOptions) (*Lake, error) {
	m, err := MetadataRead(ctx, root.AppendPath(metadataFilename))
	if err != nil {
		return nil, err
	}
	if m.DataPath == "." {
		m.DataPath = root.String()
	}
	dpuri, err := iosrc.ParseURI(m.DataPath)
	if err != nil {
		return nil, err
	}

	lk := &Lake{
		DataOrder:        m.DataOrder,
		DataPath:         dpuri,
		LogSizeThreshold: m.LogSizeThreshold,
		Root:             root,
		immfiles:         iosrc.DefaultMuxSource,
	}

	if oo != nil && oo.ImmutableCache != nil {
		lk.immfiles = oo.ImmutableCache
	}

	return lk, nil
}

func CreateOrOpenLake(rpath string, co *CreateOptions, oo *OpenOptions) (*Lake, error) {
	return CreateOrOpenLakeWithContext(context.Background(), rpath, co, oo)
}

func CreateOrOpenLakeWithContext(ctx context.Context, rpath string, co *CreateOptions, oo *OpenOptions) (*Lake, error) {
	root, err := iosrc.ParseURI(rpath)
	if err != nil {
		return nil, err
	}

	mdPath := root.AppendPath(metadataFilename)
	ok, err := iosrc.Exists(ctx, mdPath)
	if err != nil {
		// The error encountered here has been an S3 permission failure,
		// and it is fatal, so a panic might be better.
		// A log message would be good in any case. Providing access to
		// a logger will require modifying mulitple layers of APIs, so we
		// should discuss the best solution first. -Mark
		return nil, err
	}
	if !ok {
		if err := iosrc.MkdirAll(root.AppendPath(dataDirname), 0700); err != nil {
			return nil, err
		}
		if co == nil {
			co = &CreateOptions{}
		}
		if err := co.toMetadata().Write(mdPath); err != nil {
			return nil, err
		}
	}

	return openLake(ctx, root, oo)
}
