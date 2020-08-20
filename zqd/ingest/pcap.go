package ingest

import (
	"bufio"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/pcapstorage"
	"github.com/brimsec/zq/zqd/zeek"
)

type ClearableLogStore interface {
	LogStore
	Clear(ctx context.Context) error
}

type PcapOp struct {
	StartTime nano.Ts
	PcapSize  int64

	pcapstore    *pcapstorage.Store
	logstore     ClearableLogStore
	snapshots    int32
	pcapuri      iosrc.URI
	pcapReadSize int64
	logdir       string
	done, snap   chan struct{}
	err          error
	zlauncher    zeek.Launcher
}

// NewPcapOp kicks of the process for ingesting a pcap file into a space.
// Should everything start out successfully, this will return a thread safe
// Process instance once zeek log files have started to materialize in a tmp
// directory. If zeekExec is an empty string, this will attempt to resolve zeek
// from $PATH.
func NewPcapOp(ctx context.Context, pcapstore *pcapstorage.Store, logstore ClearableLogStore, pcap string, zlauncher zeek.Launcher) (*PcapOp, error) {
	pcapuri, err := iosrc.ParseURI(pcap)
	if err != nil {
		return nil, err
	}
	info, err := iosrc.Stat(pcapuri)
	if err != nil {
		return nil, err
	}
	if err := pcapstore.Update(pcapuri); err != nil {
		return nil, err
	}
	logdir, err := ioutil.TempDir("", "zqd-pcap-ingest-")
	if err != nil {
		return nil, err
	}
	p := &PcapOp{
		StartTime: nano.Now(),
		PcapSize:  info.Size(),
		pcapstore: pcapstore,
		logstore:  logstore,
		pcapuri:   pcapuri,
		logdir:    logdir,
		done:      make(chan struct{}),
		snap:      make(chan struct{}),
		zlauncher: zlauncher,
	}
	go func() {
		p.err = p.run(ctx)
		close(p.done)
		close(p.snap)
	}()
	return p, nil
}

func (p *PcapOp) run(ctx context.Context) error {
	var slurpErr error
	slurpDone := make(chan struct{})
	go func() {
		slurpErr = p.runZeek(ctx)
		close(slurpDone)
	}()

	abort := func() {
		os.RemoveAll(p.logdir)
		// Don't want to use passed context here because a cancelled context
		// would cause storage not to be cleared.
		p.pcapstore.Delete()
		p.logstore.Clear(context.Background())
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

func (p *PcapOp) runZeek(ctx context.Context) error {
	pcapfile, err := iosrc.NewReader(p.pcapuri)
	if err != nil {
		return err
	}
	defer pcapfile.Close()
	r := io.TeeReader(pcapfile, p)
	zproc, err := p.zlauncher(ctx, bufio.NewReader(r), p.logdir)
	if err != nil {
		return err
	}
	return zproc.Wait()
}

// PcapReadSize returns the total size in bytes of data read from the underlying
// pcap file.
func (p *PcapOp) PcapReadSize() int64 {
	return atomic.LoadInt64(&p.pcapReadSize)
}

// Err returns the an error if an error occurred while the ingest process was
// running. If the process is still running Err will wait for the process to
// complete before returning.
func (p *PcapOp) Err() error {
	<-p.done
	return p.err
}

// Done returns a chan that emits when the ingest process is complete.
func (p *PcapOp) Done() <-chan struct{} {
	return p.done
}

func (p *PcapOp) SnapshotCount() int {
	return int(atomic.LoadInt32(&p.snapshots))
}

// Snap returns a chan that emits every time a snapshot is made. It
// should no longer be read from after Done() has emitted.
func (p *PcapOp) Snap() <-chan struct{} {
	return p.snap
}

func (p *PcapOp) createSnapshot(ctx context.Context) error {
	files, err := filepath.Glob(filepath.Join(p.logdir, "*.log"))
	// Per filepath.Glob documentation the only possible error would be due to
	// an invalid glob pattern. Ok to panic.
	if err != nil {
		panic(err)
	}
	if len(files) == 0 {
		return nil
	}
	// convert logs into sorted zng
	zctx := resolver.NewContext()
	zr, err := detector.OpenFiles(zctx, zbuf.RecordCompare(p.logstore.NativeDirection()), files...)
	if err != nil {
		return err
	}
	defer zr.Close()
	if err := p.logstore.Rewrite(ctx, zctx, zr); err != nil {
		return err
	}
	atomic.AddInt32(&p.snapshots, 1)
	return nil
}

func (p *PcapOp) Write(b []byte) (int, error) {
	n := len(b)
	atomic.AddInt64(&p.pcapReadSize, int64(n))
	return n, nil
}
