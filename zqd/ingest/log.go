package ingest

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqe"
	"github.com/brimsec/zq/zql"
	"go.uber.org/zap"
)

const allZngTmpFile = space.AllZngFile + ".tmp"

// Logs ingests the provided list of files into the provided space.
// Like ingest.Pcap, this overwrites any existing data in the space.
func Logs(ctx context.Context, pipe *api.JSONPipe, s *space.Space, req api.LogPostRequest) error {
	ingestDir := s.DataPath(tmpIngestDir)
	if err := os.Mkdir(ingestDir, 0700); err != nil {
		// could be in use by pcap or log ingest
		if os.IsExist(err) {
			return ErrIngestProcessInFlight
		}
		return err
	}
	defer os.RemoveAll(ingestDir)

	if err := pipe.Send(&api.TaskStart{"TaskStart", 0}); err != nil {
		verr := &api.Error{Type: "INTERNAL", Message: err.Error()}
		return pipe.SendFinal(&api.TaskEnd{"TaskEnd", 0, verr})
	}
	if err := ingestLogs(ctx, pipe, s, req); err != nil {
		os.Remove(s.DataPath(space.AllZngFile))
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

type readCounter struct {
	f     *os.File
	nread int64
}

func (rc *readCounter) Read(p []byte) (int, error) {
	n, err := rc.f.Read(p)
	rc.nread += int64(n)
	return n, err
}

func (rc *readCounter) Close() error {
	return rc.f.Close()
}

func openIncomingLog(path string) (*readCounter, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	if info.IsDir() {
		return nil, 0, zqe.E(zqe.Invalid, "path is a directory")
	}
	f, err := fs.Open(path)
	if err != nil {
		return nil, 0, err
	}
	return &readCounter{f: f}, info.Size(), nil
}

func ingestLogs(ctx context.Context, pipe *api.JSONPipe, s *space.Space, req api.LogPostRequest) error {
	zctx := resolver.NewContext()
	var readers []zbuf.Reader
	defer func() {
		for _, r := range readers {
			if closer, ok := r.(io.Closer); ok {
				closer.Close()
			}
		}
	}()
	cfg := detector.OpenConfig{}
	if req.JSONTypeConfig != nil {
		cfg.JSONTypeConfig = req.JSONTypeConfig
		cfg.JSONPathRegex = DefaultJSONPathRegexp
	}
	var totalSize int64
	var readCounters []*readCounter
	for _, path := range req.Paths {
		rc, size, err := openIncomingLog(path)
		if err != nil {
			return err
		}
		sf, err := detector.OpenFromNamedReadCloser(zctx, rc, path, cfg)
		if err != nil {
			if req.StopErr {
				return fmt.Errorf("%s: %w", path, err)
			}
			pipe.Send(&api.LogPostWarning{
				Type:    "LogPostWarning",
				Warning: fmt.Sprintf("%s: %s", path, err),
			})
			continue
		}

		totalSize += size
		readCounters = append(readCounters, rc)
		readers = append(readers, sf)
	}

	logBytesRead := func() int64 {
		var n int64
		for i := range readCounters {
			n += readCounters[i].nread
		}
		return n
	}

	zngfile, err := s.CreateFile(filepath.Join(tmpIngestDir, allZngTmpFile))
	if err != nil {
		return err
	}
	zw := zngio.NewWriter(zngfile, zio.WriterFlags{StreamRecordsMax: s.StreamSize()})
	const program = "sort -r ts | (filter *; head 1; tail 1)"
	var headW, tailW recWriter

	mux, err := compileLogIngest(ctx, s, readers, program, req.StopErr)
	if err != nil {
		return err
	}
	d := &logdriver{
		pipe:         pipe,
		startTime:    nano.Now(),
		totalSize:    totalSize,
		logBytesRead: logBytesRead,
		writers:      []zbuf.Writer{zw, &headW, &tailW},
	}

	statsTicker := time.NewTicker(search.StatsInterval)
	defer statsTicker.Stop()
	err = driver.Run(mux, d, statsTicker.C)
	if err != nil {
		zngfile.Close()
		os.Remove(zngfile.Name())
		return err
	}
	if err := zngfile.Close(); err != nil {
		os.Remove(zngfile.Name())
		return err
	}
	if tailW.r != nil {
		min := nano.Min(tailW.r.Ts, headW.r.Ts)
		max := nano.Max(tailW.r.Ts, headW.r.Ts)
		if err = s.SetSpan(nano.NewSpanTs(min, max+1)); err != nil {
			os.Remove(zngfile.Name())
			return err
		}
	}
	if err := os.Rename(zngfile.Name(), s.DataPath(space.AllZngFile)); err != nil {
		os.Remove(zngfile.Name())
		return err
	}
	return pipe.Send(&api.LogPostStatus{
		Type:         "LogPostStatus",
		LogTotalSize: d.totalSize,
		LogReadSize:  logBytesRead(),
	})
}

func compileLogIngest(ctx context.Context, s *space.Space, rs []zbuf.Reader, prog string, stopErr bool) (*driver.MuxOutput, error) {
	p, err := zql.ParseProc(prog)
	if err != nil {
		return nil, err
	}
	if stopErr {
		r := zbuf.NewCombiner(rs)
		return driver.Compile(ctx, p, r, false, nano.MaxSpan, zap.NewNop())
	}
	wch := make(chan string, 5)
	for i, r := range rs {
		rs[i] = zbuf.NewWarningReader(r, wch)
	}
	r := zbuf.NewCombiner(rs)
	return driver.CompileWarningsCh(ctx, p, r, false, nano.MaxSpan, zap.NewNop(), wch)
}
