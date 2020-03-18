package packet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
)

var (
	ErrIngestProcessInFlight = errors.New("another ingest process is already in flight for this space")
)

const (
	IndexFile        = "packets.idx.json"
	DefaultSortLimit = 10000000
)

type IngestProcess struct {
	StartTime nano.Ts
	PcapSize  int64

	space        *space.Space
	sortLimit    int
	pcapPath     string
	pcapReadSize int64
	logdir       string
	done, snap   chan struct{}
	err          error
	zeekExec     string
}

// IngestFile kicks of the process for ingesting a pcap file into a space.
// Should everything start out successfully, this will return a thread safe
// IngestProcess instance once zeek log files have started to materialize in a tmp
// directory. If zeekExec is an empty string, this will attempt to resolve zeek
// from $PATH.
func IngestFile(ctx context.Context, s *space.Space, pcap, zeekExec string, sortLimit int) (*IngestProcess, error) {
	logdir := s.DataPath(".tmp.zeeklogs")
	if err := os.Mkdir(logdir, 0700); err != nil {
		if os.IsExist(err) {
			return nil, ErrIngestProcessInFlight
		}
		return nil, err
	}
	if sortLimit == 0 {
		sortLimit = DefaultSortLimit
	}
	info, err := os.Stat(pcap)
	if err != nil {
		return nil, err
	}
	if zeekExec == "" {
		zeekExec = "zeek"
	}
	zeekExec, err = exec.LookPath(zeekExec)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("zeek not found")
		}
		return nil, fmt.Errorf("error starting zeek: %w", err)
	}
	p := &IngestProcess{
		StartTime: nano.Now(),
		PcapSize:  info.Size(),
		space:     s,
		pcapPath:  pcap,
		logdir:    logdir,
		done:      make(chan struct{}),
		snap:      make(chan struct{}),
		zeekExec:  zeekExec,
		sortLimit: sortLimit,
	}
	if err = p.indexPcap(); err != nil {
		os.Remove(p.space.DataPath(IndexFile))
		return nil, err
	}
	if err = p.space.SetPacketPath(p.pcapPath); err != nil {
		os.Remove(p.space.DataPath(IndexFile))
		return nil, err
	}
	go func() {
		p.err = p.run(ctx)
		close(p.done)
		close(p.snap)
	}()
	return p, nil
}

func (p *IngestProcess) run(ctx context.Context) error {
	var slurpErr error
	slurpDone := make(chan struct{})
	go func() {
		slurpErr = p.slurp(ctx)
		close(slurpDone)
	}()

	abort := func() {
		os.RemoveAll(p.logdir)
		os.Remove(p.space.DataPath(IndexFile))
		os.Remove(p.space.DataPath("all.bzng"))
		p.space.SetPacketPath("")
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	start := time.Now()
	next := time.Second
outer:
	for {
		select {
		case <-slurpDone:
			break outer
		case t := <-ticker.C:
			if t.After(start.Add(next)) {
				if err := p.createSnapshot(ctx); err != nil {
					abort()
					return err
				}
				select {
				case p.snap <- struct{}{}:
				default:
				}
				next = 2 * next
			}
		}
	}
	if slurpErr != nil {
		abort()
		return slurpErr
	}
	if err := p.createSnapshot(ctx); err != nil {
		abort()
		return err
	}
	if err := os.RemoveAll(p.logdir); err != nil {
		abort()
		return err
	}
	return nil
}

func (p *IngestProcess) indexPcap() error {
	pcapfile, err := os.Open(p.pcapPath)
	if err != nil {
		return err
	}
	defer pcapfile.Close()
	idx, err := pcap.CreateIndex(pcapfile, 10000)
	if err != nil {
		return err
	}
	idxpath := p.space.DataPath(IndexFile)
	tmppath := idxpath + ".tmp"
	f, err := os.Create(tmppath)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(idx); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmppath, idxpath)
}

func (p *IngestProcess) slurp(ctx context.Context) error {
	pcapfile, err := os.Open(p.pcapPath)
	if err != nil {
		return err
	}
	defer pcapfile.Close()
	zeekproc, zeekwriter, err := p.startZeek(ctx, p.logdir)
	if err != nil {
		return err
	}
	w := io.MultiWriter(zeekwriter, p)
	if _, err := io.Copy(w, pcapfile); err != nil {
		return err
	}
	// Now that all data has been copied over, close stdin for zeek process so
	// process gracefully exits.
	if err := zeekwriter.Close(); err != nil {
		return err
	}
	if err := zeekproc.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := zeekproc.Stderr.(*bytes.Buffer).String()
			stderr = strings.TrimSpace(stderr)
			return fmt.Errorf("zeek exited with status %d: %s", exitErr.ExitCode(), stderr)
		}
		return err
	}
	return nil
}

// PcapReadSize returns the total size in bytes of data read from the underlying
// pcap file.
func (p *IngestProcess) PcapReadSize() int64 {
	return atomic.LoadInt64(&p.pcapReadSize)
}

// Err returns the an error if an error occurred while the ingest process was
// running. If the process is still running Err will wait for the process to
// complete before returning.
func (p *IngestProcess) Err() error {
	<-p.done
	return p.err
}

// Done returns a chan that emits when the ingest process is complete.
func (p *IngestProcess) Done() <-chan struct{} {
	return p.done
}

// Snap returns a chan that emits every time a snapshot is made. It
// should no longer be read from after Done() has emitted.
func (p *IngestProcess) Snap() <-chan struct{} {
	return p.snap
}

type recWriter struct {
	r *zng.Record
}

func (rw *recWriter) Write(r *zng.Record) error {
	rw.r = r
	return nil
}

func (p *IngestProcess) createSnapshot(ctx context.Context) error {
	files, err := filepath.Glob(filepath.Join(p.logdir, "*.log"))
	// Per filepath.Glob documentation the only possible error would be due to
	// an invalid glob pattern. Ok to panic.
	if err != nil {
		panic(err)
	}
	if len(files) == 0 {
		return nil
	}
	// convert logs into sorted bzng
	var sortedFiles []string
	for _, f := range files {
		sorted, err := splitAndSort(ctx, f, p.sortLimit)
		if err != nil {
			return err
		}
		sortedFiles = append(sortedFiles, sorted...)
	}
	zr, err := scanner.OpenFiles(resolver.NewContext(), sortedFiles...)
	if err != nil {
		return err
	}
	defer zr.Close()
	// For the time being, this endpoint will overwrite any underlying data.
	// In order to get rid errors on any concurrent searches on this space,
	// write bzng to a temp file and rename on successful conversion.
	bzngfile, err := p.space.CreateFile("all.bzng.tmp")
	if err != nil {
		return err
	}
	zw := bzngio.NewWriter(bzngfile)
	var headW, tailW recWriter
	const program = "(filter *; head 1; tail 1)"
	if err := search.Copy(ctx, []zbuf.Writer{zw, &headW, &tailW}, zr, program); err != nil {
		// If an error occurs here close and remove tmp bzngfile, lest we start
		// leaking files and file descriptors.
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
		if err = p.space.SetTimes(minTs, maxTs); err != nil {
			return err
		}
	}
	return os.Rename(bzngfile.Name(), p.space.DataPath("all.bzng"))
}

func (p *IngestProcess) startZeek(ctx context.Context, dir string) (*exec.Cmd, io.WriteCloser, error) {
	const disable = `event zeek_init() { Log::disable_stream(PacketFilter::LOG); Log::disable_stream(LoadedScripts::LOG); }`
	cmd := exec.CommandContext(ctx, p.zeekExec, "-C", "-r", "-", "--exec", disable, "local")
	cmd.Dir = dir
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	// Capture stderr for error reporting.
	cmd.Stderr = bytes.NewBuffer(nil)
	return cmd, w, cmd.Start()
}

func (p *IngestProcess) Write(b []byte) (int, error) {
	n := len(b)
	atomic.AddInt64(&p.pcapReadSize, int64(n))
	return n, nil
}

// splitAndSort splits records in filename into two BZNG files sorted by
// increasing timestamp.  Records already in this order are written directly to
// one file; other records are sorted in memory and written to the other file.
// Names of the new files are returned in a slice.  An error is returned if more
// than sortLimit records must be sorted in memory.
func splitAndSort(ctx context.Context, filename string, sortLimit int) ([]string, error) {
	zr, err := scanner.OpenFiles(resolver.NewContext(), filename)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	f1, err := ioutil.TempFile(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	defer f1.Close()
	f2, err := ioutil.TempFile(filepath.Dir(filename), filepath.Base(filename))
	if err != nil {
		return nil, err
	}
	defer f2.Close()
	alreadySorted := bzngio.NewWriter(f1)
	notYetSorted := make(recordChannelReader)
	errCh := make(chan error)
	go func() {
		program := fmt.Sprintf("sort -limit %d ts", sortLimit)
		errCh <- search.Copy(ctx, []zbuf.Writer{bzngio.NewWriter(f2)}, notYetSorted, program)
	}()
	var cur nano.Ts
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rec, err := zr.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if cur <= rec.Ts {
			cur = rec.Ts
			if err := alreadySorted.Write(rec); err != nil {
				return nil, err
			}
		} else {
			select {
			case notYetSorted <- rec:
			case <-ctx.Done():
				return nil, err
			}
		}
	}
	close(notYetSorted)
	if err := <-errCh; err != nil {
		return nil, err
	}
	if err := f1.Close(); err != nil {
		return nil, err
	}
	if err := f2.Close(); err != nil {
		return nil, err
	}
	return []string{f1.Name(), f2.Name()}, err
}

type recordChannelReader chan *zng.Record

func (z recordChannelReader) Read() (*zng.Record, error) {
	return <-z, nil
}
