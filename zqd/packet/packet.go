package packet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"gopkg.in/fsnotify.v1"
)

var (
	ErrIngestProcessInFlight = errors.New("another ingest process is already in flight for this space")
)

const IndexFile = "packets.idx.json"

type IngestProcess struct {
	space        *space.Space
	pcapPath     string
	pcapSize     int64
	pcapReadSize int64
	logdir       string
	done         chan error
	start        nano.Ts
}

// IngestFile kicks of the process for ingesting a pcap file into a space.
// Should everything start out successfully, this will return a thread safe
// IngestProcess instance once zeek log files have started to materialize in a tmp
// directory.
func IngestFile(ctx context.Context, s *space.Space, pcap string) (*IngestProcess, error) {
	logdir := s.DataPath(".tmp.zeeklogs")
	if err := os.Mkdir(logdir, 0700); err != nil {
		if os.IsExist(err) {
			return nil, ErrIngestProcessInFlight
		}
		return nil, err
	}
	info, err := os.Stat(pcap)
	if err != nil {
		return nil, err
	}
	p := &IngestProcess{
		space:    s,
		pcapPath: pcap,
		pcapSize: info.Size(),
		logdir:   logdir,
		done:     make(chan error),
		start:    nano.Now(),
	}
	go func() {
		if err := p.run(ctx); err != nil {
			p.done <- err
		}
		close(p.done)
	}()
	return p, p.awaitZeekLogs()
}

func (p *IngestProcess) run(ctx context.Context) error {
	idx, err := p.slurp(ctx)
	if err != nil {
		return err
	}
	if err = p.writeIndexFile(idx); err != nil {
		goto abort
	}
	if err = p.space.SetPacketPath(p.pcapPath); err != nil {
		goto abort
	}
	if err = p.writeData(ctx); err != nil {
		goto abort
	}
	if err = os.RemoveAll(p.logdir); err != nil {
		goto abort
	}
	return nil
abort:
	os.RemoveAll(p.logdir)
	os.Remove(p.space.DataPath(IndexFile))
	p.space.SetPacketPath("")
	return err
}

// awaitZeekLogs waits for the first zeek logs to hit the file system. Should
// an error occur before this happens, the error will be returned.
func (p *IngestProcess) awaitZeekLogs() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	if err := w.Add(p.logdir); err != nil {
		return err
	}
	for {
		select {
		case err := <-p.done:
			return err
		case event := <-w.Events:
			if event.Op == fsnotify.Create && filepath.Ext(event.Name) == ".log" {
				return nil
			}
		case err := <-w.Errors:
			return err
		}
	}
}

func (p *IngestProcess) writeIndexFile(idx *pcap.Index) error {
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

func (p *IngestProcess) slurp(ctx context.Context) (*pcap.Index, error) {
	pcapfile, err := os.Open(p.pcapPath)
	if err != nil {
		return nil, err
	}
	defer pcapfile.Close()
	zeekproc, zeekwriter, err := p.startZeek(ctx, p.logdir)
	if err != nil {
		return nil, err
	}
	indexwriter := pcap.NewIndexWriter(10000)
	w := io.MultiWriter(zeekwriter, indexwriter, p)
	if _, err := io.Copy(w, pcapfile); err != nil {
		return nil, err
	}
	// Now that all data has been copied over, close stdin for zeek process so
	// process gracefully exits.
	if err := zeekwriter.Close(); err != nil {
		return nil, err
	}
	if err := zeekproc.Wait(); err != nil {
		return nil, err
	}
	index, err := indexwriter.Close()
	if err != nil {
		return nil, err
	}
	return index, nil
}

func (p *IngestProcess) Status(timeout <-chan time.Time) (done bool, res api.PacketPostStatus, err error) {
	select {
	case <-timeout:
	case err = <-p.done:
		done = true
	}
	res.Type = "PacketPostStatus"
	res.PacketSize = p.pcapSize
	res.PacketReadSize = atomic.LoadInt64(&p.pcapReadSize)
	res.StartTime = p.start
	res.UpdateTime = nano.Now()
	return done, res, err
}

func (p *IngestProcess) writeData(ctx context.Context) error {
	files, err := filepath.Glob(filepath.Join(p.logdir, "*.log"))
	// Per filepath.Glob documentation the only possible error would be due to
	// an invalid glob pattern. Ok to panic.
	if err != nil {
		panic(err)
	}
	// convert logs into sorted bzng
	zr, err := scanner.OpenFiles(resolver.NewContext(), files...)
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
	const program = "_path != packet_filter _path != loaded_scripts | sort -limit 10000000 ts"
	if err := search.Copy(ctx, zw, zr, program); err != nil {
		// If an error occurs here close and remove tmp bzngfile, lest we start
		// leaking files and file descriptors.
		bzngfile.Close()
		os.Remove(bzngfile.Name())
		return nil
	}
	if err := bzngfile.Close(); err != nil {
		return err
	}
	return os.Rename(bzngfile.Name(), p.space.DataPath("all.bzng"))
}

func (p *IngestProcess) startZeek(ctx context.Context, dir string) (*exec.Cmd, io.WriteCloser, error) {
	cmd := exec.CommandContext(ctx, "zeek", "-C", "-r", "-")
	cmd.Dir = dir
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	return cmd, w, cmd.Start()
}

func (p *IngestProcess) Write(b []byte) (int, error) {
	n := len(b)
	atomic.AddInt64(&p.pcapReadSize, int64(n))
	return n, nil
}
