package archive

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/archive/immcache"
	"github.com/brimsec/zq/ppl/archive/index"
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

type Archive struct {
	Root             iosrc.URI
	DataPath         iosrc.URI
	DataOrder        zbuf.Order
	LogSizeThreshold int64
	LogFilter        []ksuid.KSUID

	immfiles interface {
		ReadFile(context.Context, iosrc.URI) ([]byte, error)
	}
}

func (ark *Archive) metaWrite() error {
	m := &Metadata{
		Version:          0,
		LogSizeThreshold: ark.LogSizeThreshold,
		DataOrder:        ark.DataOrder,
		DataPath:         ark.DataPath.String(),
	}
	return m.Write(ark.mdURI())
}

func (ark *Archive) mdURI() iosrc.URI {
	return ark.Root.AppendPath(metadataFilename)
}

func (ark *Archive) filterAllowed(id ksuid.KSUID) bool {
	if len(ark.LogFilter) == 0 {
		return true
	}
	for _, fid := range ark.LogFilter {
		if fid == id {
			return true
		}
	}
	return false
}

func (ark *Archive) DefinitionsDir() iosrc.URI {
	return ark.Root.AppendPath(indexdefsDir)
}

func (ark *Archive) ReadDefinitions(ctx context.Context) (index.Definitions, error) {
	defs, err := index.ReadDefinitions(ctx, ark.DefinitionsDir())
	if zqe.IsNotFound(err) {
		err = iosrc.MkdirAll(ark.DefinitionsDir(), 0700)
	}
	return defs, err
}

type OpenOptions struct {
	ImmutableCache immcache.ImmutableCache
}

func OpenArchive(rpath string, oo *OpenOptions) (*Archive, error) {
	return OpenArchiveWithContext(context.Background(), rpath, oo)
}

func OpenArchiveWithContext(ctx context.Context, rpath string, oo *OpenOptions) (*Archive, error) {
	root, err := iosrc.ParseURI(rpath)
	if err != nil {
		return nil, err
	}
	return openArchive(ctx, root, oo)
}

func openArchive(ctx context.Context, root iosrc.URI, oo *OpenOptions) (*Archive, error) {
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

	ark := &Archive{
		DataOrder:        m.DataOrder,
		DataPath:         dpuri,
		LogSizeThreshold: m.LogSizeThreshold,
		Root:             root,
		immfiles:         iosrc.DefaultMuxSource,
	}

	if oo != nil && oo.ImmutableCache != nil {
		ark.immfiles = oo.ImmutableCache
	}

	return ark, nil
}

func CreateOrOpenArchive(rpath string, co *CreateOptions, oo *OpenOptions) (*Archive, error) {
	return CreateOrOpenArchiveWithContext(context.Background(), rpath, co, oo)
}

func CreateOrOpenArchiveWithContext(ctx context.Context, rpath string, co *CreateOptions, oo *OpenOptions) (*Archive, error) {
	root, err := iosrc.ParseURI(rpath)
	if err != nil {
		return nil, err
	}

	mdPath := root.AppendPath(metadataFilename)
	ok, err := iosrc.Exists(ctx, mdPath)
	if err != nil {
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

	return openArchive(ctx, root, oo)
}
