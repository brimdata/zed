package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
)

const allBzngTmpFile = space.AllBzngFile + ".tmp"

// Logs ingests the provided list of files into the provided space.
// Like ingest.Pcap, this overwrites any existing data in the space.
func Logs(ctx context.Context, s *space.Space, paths []string, tc *ndjsonio.TypeConfig, sortLimit int) error {
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
	if err := ingestLogs(ctx, s, paths, tc, sortLimit); err != nil {
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

// x509.14:00:00-15:00:00.log.gz (open source zeek)
// x509_20191101_14:00:00-15:00:00+0000.log.gz (corelight)
const DefaultJSONPathRegexp = `([a-zA-Z0-9_]+)(?:\.|_\d{8}_)\d\d:\d\d:\d\d\-\d\d:\d\d:\d\d(?:[+\-]\d{4})?\.log(?:$|\.gz)`

func configureJSONTypeReader(ndjr *ndjsonio.Reader, tc ndjsonio.TypeConfig, filename string) error {
	var path string
	re := regexp.MustCompile(DefaultJSONPathRegexp)
	match := re.FindStringSubmatch(filename)
	if len(match) == 2 {
		path = match[1]
	}
	if err := ndjr.ConfigureTypes(tc, path); err != nil {
		return err
	}
	return nil
}

func ingestLogs(ctx context.Context, s *space.Space, paths []string, tc *ndjsonio.TypeConfig, sortLimit int) error {
	zctx := resolver.NewContext()
	var readers []zbuf.Reader
	for _, path := range paths {
		sf, err := scanner.OpenFile(zctx, path, "auto")
		if err != nil {
			return err
		}
		jr, ok := sf.Reader.(*ndjsonio.Reader)
		if ok && tc != nil {
			if err = configureJSONTypeReader(jr, *tc, path); err != nil {
				return err
			}
		}
		readers = append(readers, sf)
	}
	reader := scanner.NewCombiner(readers)
	defer reader.Close()

	bzngfile, err := s.CreateFile(filepath.Join(tmpIngestDir, allBzngTmpFile))
	if err != nil {
		return err
	}
	zw := bzngio.NewWriter(bzngfile)
	program := fmt.Sprintf("sort -limit %d -r ts | (filter *; head 1; tail 1)", sortLimit)
	var headW, tailW recWriter
	if err := search.Copy(ctx, []zbuf.Writer{zw, &headW, &tailW}, reader, program); err != nil {
		bzngfile.Close()
		os.Remove(bzngfile.Name())
		return err
	}
	if err := bzngfile.Close(); err != nil {
		return err
	}
	if tailW.r != nil {
		if err = s.SetTimes(tailW.r.Ts, headW.r.Ts); err != nil {
			return err
		}
	}
	return os.Rename(bzngfile.Name(), s.DataPath(space.AllBzngFile))
}
