package archive

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqe"
	"github.com/segmentio/ksuid"
)

const metadataFilename = "zar.json"

type Metadata struct {
	Version           int            `json:"version"`
	DataPath          string         `json:"data_path"`
	LogSizeThreshold  int64          `json:"log_size_threshold"`
	DataSortDirection zbuf.Direction `json:"data_sort_direction"`
}

func (c *Metadata) Write(uri iosrc.URI) error {
	src, err := iosrc.GetSource(uri)
	if err != nil {
		return err
	}
	rep, ok := src.(iosrc.ReplacerAble)
	if !ok {
		return zqe.E("scheme does not support metadata updates: %s", uri)
	}
	// Pass background context because we don't want this to quit.
	wc, err := rep.NewReplacer(context.Background(), uri)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(wc).Encode(c); err != nil {
		wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
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
	DefaultLogSizeThreshold  = 500 * 1024 * 1024
	DefaultDataSortDirection = zbuf.DirTimeReverse
)

type CreateOptions struct {
	LogSizeThreshold *int64
	DataPath         string
	SortAscending    bool
}

func (c *CreateOptions) toMetadata() *Metadata {
	m := &Metadata{
		Version:           0,
		LogSizeThreshold:  DefaultLogSizeThreshold,
		DataSortDirection: DefaultDataSortDirection,
		DataPath:          ".",
	}
	if c.LogSizeThreshold != nil {
		m.LogSizeThreshold = *c.LogSizeThreshold
	}
	if c.DataPath != "" {
		m.DataPath = c.DataPath
	}
	if c.SortAscending {
		m.DataSortDirection = zbuf.DirTimeForward
	}
	return m
}

type Archive struct {
	Root              iosrc.URI
	DataPath          iosrc.URI
	DataSortDirection zbuf.Direction
	LogSizeThreshold  int64
	LogFilter         []ksuid.KSUID
	dataSrc           iosrc.Source
}

func (ark *Archive) metaWrite() error {
	m := &Metadata{
		Version:           0,
		LogSizeThreshold:  ark.LogSizeThreshold,
		DataSortDirection: ark.DataSortDirection,
		DataPath:          ark.DataPath.String(),
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

type OpenOptions struct {
	LogFilter  []string
	DataSource iosrc.Source
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
		Root:              root,
		DataSortDirection: m.DataSortDirection,
		LogSizeThreshold:  m.LogSizeThreshold,
		DataPath:          dpuri,
	}

	if oo != nil && oo.DataSource != nil {
		ark.dataSrc = oo.DataSource
	} else {
		if ark.dataSrc, err = iosrc.GetSource(dpuri); err != nil {
			return nil, err
		}
	}
	if oo != nil && len(oo.LogFilter) != 0 {
		for _, l := range oo.LogFilter {
			df, ok := dataFileNameMatch(l)
			if !ok {
				return nil, zqe.E(zqe.Invalid, "log filter %s not a data filename", l)
			}
			ark.LogFilter = append(ark.LogFilter, df.id)
		}
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
		src, err := iosrc.GetSource(root)
		if err != nil {
			return nil, err
		}
		if dm, ok := src.(iosrc.DirMaker); ok {
			if err := dm.MkdirAll(root, 0700); err != nil {
				return nil, err
			}
			if err := dm.MkdirAll(root.AppendPath(dataDirname), 0700); err != nil {
				return nil, err
			}
		}
		if err := co.toMetadata().Write(mdPath); err != nil {
			return nil, err
		}
	}

	return openArchive(ctx, root, oo)
}
