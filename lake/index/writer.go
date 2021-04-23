package index

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
)

func NewWriter(ctx context.Context, path iosrc.URI, ref Reference) (*Writer, error) {
	rwCh := make(rwChan)
	indexer, err := newIndexer(ctx, path, ref, rwCh)
	if err != nil {
		return nil, err
	}
	w := &Writer{
		Reference: ref,
		URI:       ref.ObjectPath(path),

		done:    make(chan struct{}),
		indexer: indexer,
		rwCh:    rwCh,
	}
	return w, nil
}

type Writer struct {
	Reference *Reference
	URI       iosrc.URI

	done    chan struct{}
	indexer *indexer
	once    sync.Once
	rwCh    rwChan
}

type rwChan chan *zng.Record

func (c rwChan) Read() (*zng.Record, error) {
	return <-c, nil
}

func (w *Writer) Write(rec *zng.Record) error {
	select {
	case <-w.done:
		if err := w.indexer.err.Load(); err != nil {
			return err
		}
		return errors.New("index writer closed")
	default:
		w.once.Do(w.indexer.start)
		w.rwCh <- rec
		return nil
	}
}

func (w *Writer) Close() error {
	// If once has not be called, this means a write has never been called.
	// Abort index so no file is written.
	w.once.Do(func() {
		w.indexer.index.Abort()
	})
	close(w.done)
	close(w.rwCh)
	return w.indexer.Wait()
}

func (w *Writer) Abort() {
	w.Close()
	w.indexer.index.Abort()
}

// onceError is an object that will only store an error once.
type onceError struct {
	sync.Mutex // guards following
	err        error
}

func (a *onceError) Store(err error) {
	a.Lock()
	defer a.Unlock()
	if a.err != nil {
		return
	}
	a.err = err
}
func (a *onceError) Load() error {
	a.Lock()
	defer a.Unlock()
	return a.err
}

type indexer struct {
	err     onceError
	cutter  *expr.Cutter
	fgr     zbuf.ReadCloser
	index   *index.Writer
	keyType zng.Type
	wg      sync.WaitGroup
}

func newIndexer(ctx context.Context, path iosrc.URI, ref Reference, r zbuf.Reader) (*indexer, error) {
	rule := ref.Rule
	proc, err := rule.Proc()
	if err != nil {
		return nil, err
	}

	zctx := zson.NewContext()
	conf := driver.Config{Custom: compile}
	fgr, err := driver.NewReaderWithConfig(ctx, conf, proc, zctx, r)
	if err != nil {
		return nil, err
	}

	keys := rule.Keys
	if len(keys) == 0 {
		keys = []field.Static{keyName}
	}

	opts := []index.Option{index.KeyFields(keys...)}
	if rule.Framesize > 0 {
		opts = append(opts, index.FrameThresh(rule.Framesize))
	}

	writer, err := newIndexWriter(ctx, zctx, path, ref)
	if err != nil {
		return nil, err
	}

	fields, resolvers := compiler.CompileAssignments(keys, keys)
	cutter, err := expr.NewCutter(zctx, fields, resolvers)
	if err != nil {
		return nil, err
	}

	return &indexer{
		fgr:    fgr,
		cutter: cutter,
		index:  writer,
	}, nil
}

func newIndexWriter(ctx context.Context, zctx *zson.Context, path iosrc.URI, ref Reference, opts ...index.Option) (w *index.Writer, err error) {
	for tried := false; ; {
		spath := ref.ObjectPath(path).String()
		w, err = index.NewWriterWithContext(ctx, zctx, spath, opts...)
		if err != nil {
			if zqe.IsNotFound(err) && !tried {
				// If a not found is return this is probably because the rule
				// path has not been created. Create the dir and try once more.
				if err = iosrc.MkdirAll(ref.ObjectDir(path), 0700); err == nil {
					tried = true
					continue
				}
			}
		}
		return w, err
	}
}

func (d *indexer) start() {
	d.wg.Add(1)
	go func() {
		if err := zbuf.Copy(d, d.fgr); err != nil {
			d.index.Abort()
			d.err.Store(err)
		}
		d.err.Store(d.index.Close())
		d.wg.Done()
	}()
}

func (d *indexer) Wait() error {
	d.wg.Wait()
	return d.err.Load()
}

func (d *indexer) Write(rec *zng.Record) error {
	key, err := d.cutter.Apply(rec)
	if err != nil {
		return fmt.Errorf("checking index record: %w", err)
	}
	if d.keyType == nil {
		d.keyType = key.Type
	}
	if key.Type.ID() != d.keyType.ID() {
		return fmt.Errorf("key type changed from %q to %q", d.keyType.ZSON(), key.Type.ZSON())
	}
	return d.index.Write(rec)
}
