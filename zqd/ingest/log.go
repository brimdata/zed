package ingest

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	zdriver "github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zql"
	"go.uber.org/zap"
)

const allBzngTmpFile = space.AllBzngFile + ".tmp"

// Logs ingests the provided list of files into the provided space.
// Like ingest.Pcap, this overwrites any existing data in the space.
func Logs(ctx context.Context, pipe *api.JSONPipe, s *space.Space, paths []string, tc *ndjsonio.TypeConfig, sortLimit int) error {
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

	pipe.Send(&api.TaskStart{"TaskStart", 0})
	if err := ingestLogs(ctx, pipe, s, paths, tc, sortLimit); err != nil {
		os.Remove(s.DataPath(space.AllBzngFile))
		verr := &api.Error{Type: "INTERNAL", Message: err.Error()}
		return pipe.SendFinal(&api.TaskEnd{"TaskEnd", 0, verr})
	}
	return pipe.SendFinal(&api.TaskEnd{"TaskEnd", 0, nil})
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
	return ndjr.ConfigureTypes(tc, path)
}

func ingestLogs(ctx context.Context, pipe *api.JSONPipe, s *space.Space, paths []string, tc *ndjsonio.TypeConfig, sortLimit int) error {
	zctx := resolver.NewContext()
	var readers []zbuf.Reader
	defer func() {
		for _, r := range readers {
			if closer, ok := r.(io.Closer); ok {
				closer.Close()
			}
		}
	}()
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

	bzngfile, err := s.CreateFile(filepath.Join(tmpIngestDir, allBzngTmpFile))
	if err != nil {
		return err
	}
	zw := bzngio.NewWriter(bzngfile, zio.Flags{})
	program := fmt.Sprintf("sort -limit %d -r ts | (filter *; head 1; tail 1)", sortLimit)
	var headW, tailW recWriter
	err = runLogIngest(ctx, s, reader, program, pipe, []zbuf.Writer{zw, &headW, &tailW}...)
	if err != nil {
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
	if err = os.Rename(bzngfile.Name(), s.DataPath(space.AllBzngFile)); err != nil {
		return err
	}
	info, err := s.Info()
	if err != nil {
		return err
	}
	status := api.LogPostStatus{
		Type:    "LogPostStatus",
		MinTime: info.MinTime,
		MaxTime: info.MaxTime,
		Size:    info.Size,
	}
	pipe.Send(status)
	return nil
}

func runLogIngest(ctx context.Context, s *space.Space, r zbuf.Reader, prog string, pipe *api.JSONPipe, w ...zbuf.Writer) error {
	p, err := zql.ParseProc(prog)
	if err != nil {
		return err
	}
	mux, err := zdriver.Compile(ctx, p, r, false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}
	d := &driver{
		pipe:      pipe,
		startTime: nano.Now(),
		writers:   w,
	}
	return zdriver.Run(mux, d, search.DefaultStatsInterval)
}
