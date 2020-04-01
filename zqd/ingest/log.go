package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
)

const allBzngTmpFile = space.AllBzngFile + ".tmp"

// Logs ingests the provided list of files into the provided space.
// Like ingest.Pcap, this overwrites any existing data in the space.
func Logs(ctx context.Context, s *space.Space, paths []string, sortLimit int) error {
	ingestDir := s.DataPath(tmpIngestDir)
	if err := os.Mkdir(ingestDir, 0700); err != nil {
		// could be in use by pcap or log ingest
		if os.IsExist(err) {
			return ErrIngestProcessInFlight
		}
		return err
	}
	defer os.RemoveAll(ingestDir)
	if sortLimit == 0 {
		sortLimit = DefaultSortLimit
	}
	if err := ingestLogs(ctx, s, paths, sortLimit); err != nil {
		os.Remove(s.DataPath(space.AllBzngFile))
		return err
	}
	return nil
}

type recWriter struct {
	r *zng.Record
}

func (rw *recWriter) Write(r *zng.Record) error {
	rw.r = r
	return nil
}

func ingestLogs(ctx context.Context, s *space.Space, paths []string, sortLimit int) error {
	zr, err := scanner.OpenFiles(resolver.NewContext(), paths...)
	if err != nil {
		return err
	}
	defer zr.Close()
	bzngfile, err := s.CreateFile(filepath.Join(tmpIngestDir, allBzngTmpFile))
	if err != nil {
		return err
	}
	zw := bzngio.NewWriter(bzngfile)
	program := fmt.Sprintf("sort -limit %d -r ts | (filter *; head 1; tail 1)", sortLimit)
	var headW, tailW recWriter
	if err := search.Copy(ctx, []zbuf.Writer{zw, &headW, &tailW}, zr, program); err != nil {
		bzngfile.Close()
		os.Remove(bzngfile.Name())
		return err
	}
	if err := bzngfile.Close(); err != nil {
		return err
	}
	if tailW.r != nil {
		minTs := tailW.r.Ts
		maxTs := headW.r.Ts
		if err = s.SetTimes(minTs, maxTs); err != nil {
			return err
		}
	}
	return os.Rename(bzngfile.Name(), s.DataPath(space.AllBzngFile))
}
