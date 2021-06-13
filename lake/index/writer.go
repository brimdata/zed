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
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

func NewWriter(ctx context.Context, engine storage.Engine, path *storage.URI, ref *Reference) (*Writer, error) {
	rwCh := make(rwChan)
	indexer, err := newIndexer(ctx, engine, path, ref, rwCh)
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
	URI       *storage.URI

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
	fgr     zio.ReadCloser
	index   *index.Writer
	keyType zng.Type
	wg      sync.WaitGroup
}

func newIndexer(ctx context.Context, engine storage.Engine, path *storage.URI, ref *Reference, r zio.Reader) (*indexer, error) {
	idx := ref.Index
	zedQuery, err := idx.zedQuery()
	if err != nil {
		return nil, err
	}
	p, err := compiler.ParseProc(zedQuery)
	if err != nil {
		return nil, err
	}
	zctx := zson.NewContext()
	fgr, err := driver.NewReader(ctx, p, zctx, r)
	if err != nil {
		return nil, err
	}
	keys := idx.Keys
	if len(keys) == 0 {
		keys = field.List{field.Dotted(keyName)}
	}
	opts := []index.Option{index.KeyFields(keys...)}
	if idx.Framesize > 0 {
		opts = append(opts, index.FrameThresh(idx.Framesize))
	}
	writer, err := newIndexWriter(ctx, zctx, engine, path, ref)
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

func newIndexWriter(ctx context.Context, zctx *zson.Context, engine storage.Engine, path *storage.URI, ref *Reference, opts ...index.Option) (w *index.Writer, err error) {
	op := ref.ObjectPath(path)
	return index.NewWriterWithContext(ctx, zctx, engine, op.String(), opts...)
}

func (d *indexer) start() {
	d.wg.Add(1)
	go func() {
		if err := zio.Copy(d, d.fgr); err != nil {
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
		return fmt.Errorf("key type changed from %q to %q", d.keyType, key.Type)
	}
	return d.index.Write(rec)
}
