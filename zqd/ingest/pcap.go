package ingest

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/pcapanalyzer"
	"github.com/brimsec/zq/zqd/pcapstorage"
	"github.com/brimsec/zq/zqd/storage"
)

//go:generate go run ../../pkg/jsontyper -o ./suricata.go -package ingest -var suricataTC ../../suricata/types.json

type ClearableStore interface {
	storage.Storage
	Clear(ctx context.Context) error
}

type PcapOp struct {
	StartTime nano.Ts
	PcapSize  int64

	pcapstore            *pcapstorage.Store
	store                ClearableStore
	snapshots            int32
	pcapuri              iosrc.URI
	pcapReadSize         int64
	logdir               string
	done, snap           chan struct{}
	err                  error
	slauncher, zlauncher pcapanalyzer.Launcher
}

// NewPcapOp kicks of the process for ingesting a pcap file into a space.
// Should everything start out successfully, this will return a thread safe
// Process instance once zeek log files have started to materialize in a tmp
// directory. If zeekExec is an empty string, this will attempt to resolve zeek
// from $PATH.
func NewPcapOp(ctx context.Context, pcapstore *pcapstorage.Store, store ClearableStore, pcap string, slauncher, zlauncher pcapanalyzer.Launcher) (*PcapOp, []string, error) {
	pcapuri, err := iosrc.ParseURI(pcap)
	if err != nil {
		return nil, nil, err
	}
	if slauncher == nil && zlauncher == nil {
		return nil, nil, fmt.Errorf("must provide at least one launcher")
	}
	info, err := iosrc.Stat(ctx, pcapuri)
	if err != nil {
		return nil, nil, err
	}
	warn := make(chan string)
	go func() {
		err = pcapstore.Update(ctx, pcapuri, warn)
		close(warn)
	}()
	var warnings []string
	for w := range warn {
		warnings = append(warnings, w)
	}
	if err != nil {
		return nil, warnings, err
	}
	logdir, err := ioutil.TempDir("", "zqd-pcap-ingest-")
	if err != nil {
		return nil, warnings, err
	}
	p := &PcapOp{
		StartTime: nano.Now(),
		PcapSize:  info.Size(),
		pcapstore: pcapstore,
		store:     store,
		pcapuri:   pcapuri,
		logdir:    logdir,
		done:      make(chan struct{}),
		snap:      make(chan struct{}),
		slauncher: slauncher,
		zlauncher: zlauncher,
	}
	go func() {
		p.err = p.run(ctx)
		close(p.done)
		close(p.snap)
	}()
	return p, warnings, nil
}

func (p *PcapOp) run(ctx context.Context) error {
	var sErr, zErr error
	var wg sync.WaitGroup
	slurpDone := make(chan struct{})
	if p.slauncher != nil {
		wg.Add(1)
		go func() {
			sErr = p.runSuricata(ctx)
			wg.Done()
		}()
	}
	if p.zlauncher != nil {
		wg.Add(1)
		go func() {
			zErr = p.runZeek(ctx)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(slurpDone)
	}()

	abort := func() {
		os.RemoveAll(p.logdir)
		// Don't want to use passed context here because a cancelled context
		// would cause storage not to be cleared.
		p.pcapstore.Delete(context.Background())
		p.store.Clear(context.Background())
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
	if sErr != nil {
		abort()
		return sErr
	}
	if zErr != nil {
		abort()
		return zErr
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
	pcapfile, err := iosrc.NewReader(ctx, p.pcapuri)
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

func (p *PcapOp) runSuricata(ctx context.Context) error {
	pcapfile, err := iosrc.NewReader(ctx, p.pcapuri)
	if err != nil {
		return err
	}
	defer pcapfile.Close()
	sproc, err := p.slauncher(ctx, bufio.NewReader(pcapfile), p.logdir)
	if err != nil {
		return err
	}
	if err = sproc.Wait(); err != nil {
		return err
	}
	return p.convertSuricataLog(ctx)
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

func (p *PcapOp) zeekFiles() []string {
	files, err := filepath.Glob(filepath.Join(p.logdir, "*.log"))
	// Per filepath.Glob documentation the only possible error would be due to
	// an invalid glob pattern. Ok to panic.
	if err != nil {
		panic(err)
	}
	return files
}

func (p *PcapOp) suricataFiles() []string {
	path := filepath.Join(p.logdir, "eve.zng")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return []string{path}
}

func (p *PcapOp) createSnapshot(ctx context.Context) error {
	files := append(p.zeekFiles(), p.suricataFiles()...)
	if len(files) == 0 {
		return nil
	}
	// convert logs into sorted zng
	zctx := resolver.NewContext()
	zr, err := detector.OpenFiles(ctx, zctx, zbuf.RecordCompare(p.store.NativeDirection()), files...)
	if err != nil {
		return err
	}
	defer zr.Close()
	if err := p.store.Write(ctx, zctx, zr); err != nil {
		return err
	}
	atomic.AddInt32(&p.snapshots, 1)
	return nil
}

func (p *PcapOp) convertSuricataLog(ctx context.Context) error {
	// For now, this just converts from json to
	// zng. Soon it will apply a typing config and rename at least
	// the timestamp field.
	zctx := resolver.NewContext()
	path := filepath.Join(p.logdir, "eve.json")
	zr, err := detector.OpenFile(zctx, path, zio.ReaderOpts{})
	if err != nil {
		return err
	}
	defer zr.Close()
	return fs.ReplaceFile(filepath.Join(p.logdir, "eve.zng"), os.FileMode(0666), func(w io.Writer) error {
		zw := zngio.NewWriter(zio.NopCloser(w), zngio.WriterOpts{})
		return zbuf.CopyWithContext(ctx, zw, zr)
	})
}

func (p *PcapOp) Write(b []byte) (int, error) {
	n := len(b)
	atomic.AddInt64(&p.pcapReadSize, int64(n))
	return n, nil
}
