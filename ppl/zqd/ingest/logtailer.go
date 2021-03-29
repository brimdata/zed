package ingest

import (
	"path/filepath"
	"sync"

	"github.com/brimdata/zq/pkg/fs"
	"github.com/brimdata/zq/zio"
	"github.com/brimdata/zq/zio/detector"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

type result struct {
	rec *zng.Record
	err error
}

// logTailer is a zbuf.Reader that watches a specified directory and starts
// tailing existing and newly created files in the directory for new logs. Newly
// written log data are transformed into *zng.Records and returned on a
// first-come-first serve basis.
type logTailer struct {
	opts    zio.ReaderOpts
	readers map[string]*fs.TFile
	watcher *fs.DirWatcher
	zctx    *resolver.Context

	// synchronization primitives
	results chan result
	once    sync.Once
	wg      sync.WaitGroup
}

func newLogTailer(zctx *resolver.Context, dir string, opts zio.ReaderOpts) (*logTailer, error) {
	dir = filepath.Clean(dir)
	watcher, err := fs.NewDirWatcher(dir)
	if err != nil {
		return nil, err
	}
	r := &logTailer{
		opts:    opts,
		readers: make(map[string]*fs.TFile),
		results: make(chan result, 5),
		watcher: watcher,
		zctx:    zctx,
	}
	return r, nil
}

func (d *logTailer) start() {
	var err error
	for {
		ev, ok := <-d.watcher.Events
		// Watcher closed. Enstruct all go routines to stop tailing files so
		// they read remaining data then exit.
		if !ok {
			d.stopReaders(false)
			break
		}
		if ev.Err != nil {
			err = ev.Err
			d.stopReaders(true)
			break
		}
		if ev.Op.Exists() {
			if terr := d.tailFile(ev.Name); terr != nil {
				err = terr
				d.stopReaders(true)
				break
			}
		}
	}
	// Wait for all tail go routines to stop. We are about to close the results
	// channel and do not want a write to closed channel panic.
	d.wg.Wait()
	// signfy EOS and close channel
	d.results <- result{err: err}
	close(d.results)
}

// stopReaders instructs all open TFile to stop tailing their respective files.
// If close is set to false, the readers will read through the remaining data
// in their files before emitting EOF. If close is set to true, the file
// descriptors will be closed and no further data will be read.
func (d *logTailer) stopReaders(close bool) {
	for _, r := range d.readers {
		if close {
			r.Close()
		}
		r.Stop()
	}
}

func (d *logTailer) tailFile(file string) error {
	if _, ok := d.readers[file]; ok {
		return nil
	}
	f, err := fs.TailFile(file)
	if err == fs.ErrIsDir {
		return nil
	}
	if err != nil {
		return err
	}
	d.readers[file] = f
	d.wg.Add(1)
	go func() {
		zr, err := detector.OpenFromNamedReadCloser(d.zctx, f, file, d.opts)
		if err != nil {
			d.results <- result{err: err}
			return
		}
		var res result
		for {
			res.rec, res.err = zr.Read()
			if res.rec != nil || res.err != nil {
				d.results <- res
			}
			if res.rec == nil || res.err != nil {
				d.wg.Done()
				return
			}
		}
	}()
	return nil
}

func (d *logTailer) Read() (*zng.Record, error) {
	d.once.Do(func() { go d.start() })
	res, ok := <-d.results
	if !ok {
		// already closed return EOS
		return nil, nil
	}
	if res.err != nil {
		d.watcher.Stop() // exits loop
		// drain results
		for range d.results {
		}
	}
	return res.rec, res.err
}

// Stop instructs the directory watcher and indiviual file watchers to stop
// watching for changes. Read() will emit EOS when the remaining unread data
// in files has been read.
func (d *logTailer) Stop() error {
	return d.watcher.Stop()
}
