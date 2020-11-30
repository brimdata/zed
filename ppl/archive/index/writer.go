package index

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/compiler"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"gopkg.in/tomb.v2"
)

func NewWriter(ctx context.Context, u iosrc.URI, def *Def) (*Writer, error) {
	indexer, err := newIndexer(ctx, u, def)
	if err != nil {
		return nil, err
	}
	w := &Writer{
		URI:     u,
		indexer: indexer,
	}
	w.WriteCloser = zbuf.BatchRecordWriter(w)
	return w, nil
}

type Writer struct {
	URI iosrc.URI
	zbuf.WriteCloser

	indexer *indexer
	once    sync.Once
}

func (w *Writer) WriteBatch(batch zbuf.Batch) error {
	select {
	case w.indexer.wch <- batch:
		return nil
	case <-w.indexer.Dying():
		return errors.New("writer closed")
	}
}

func (w *Writer) Write(rec *zng.Record) error {
	select {
	case <-w.indexer.Dying():
		return errors.New("writer closed")
	default:
		return w.WriteCloser.Write(rec)
	}
}

func (w *Writer) Close() error {
	err := w.WriteCloser.Close()
	w.indexer.Kill(nil)
	if ierr := w.indexer.Wait(); err == nil && ierr != context.Canceled {
		err = ierr
	}
	return err
}

func (w *Writer) Abort() {
	w.Close()
	w.indexer.microindex.Abort()
}

type indexer struct {
	tomb.Tomb
	cutter     *expr.Cutter
	keyType    zng.Type
	leaf       proc.Interface
	microindex *microindex.Writer
	wch        chan zbuf.Batch
}

func newIndexer(ctx context.Context, u iosrc.URI, def *Def) (*indexer, error) {
	keys := def.Keys
	if len(keys) == 0 {
		keys = []field.Static{keyName}
	}
	zctx := resolver.NewContext()
	opts := []microindex.Option{microindex.KeyFields(keys...)}
	if def.Framesize > 0 {
		opts = append(opts, microindex.FrameThresh(def.Framesize))
	}
	writer, err := microindex.NewWriterWithContext(ctx, zctx, u.String(), opts...)
	if err != nil {
		return nil, err
	}
	fields, resolvers := expr.CompileAssignments(keys, keys)
	d := &indexer{
		cutter:     expr.NewStrictCutter(zctx, false, fields, resolvers),
		microindex: writer,
		wch:        make(chan zbuf.Batch),
	}
	pctx := &proc.Context{Context: context.Background(), TypeContext: zctx}
	leaves, err := compiler.Compile(compile, def.Proc, pctx, []proc.Interface{d})
	if err != nil {
		return nil, err
	}
	if len(leaves) != 1 {
		return nil, fmt.Errorf("flowgraph can only have 1 leaf, has %d", len(leaves))
	}
	d.leaf = leaves[0]
	d.Go(d.run)
	return d, nil
}

func (h *indexer) Done() {
	// XXX not sure what I'm supposed to do here.
}

func (h *indexer) Pull() (zbuf.Batch, error) {
	select {
	case <-h.Dying():
		return nil, h.Err()
	case batch := <-h.wch:
		return batch, nil
	}
}

func (w *indexer) run() error {
	if err := zbuf.CopyPuller(w, w.leaf); err != nil {
		w.microindex.Abort()
		return err
	}
	return w.microindex.Close()
}

func (d *indexer) Write(rec *zng.Record) error {
	key, err := d.cutter.Cut(rec)
	if err != nil {
		return fmt.Errorf("checking index record: %w", err)
	}
	if d.keyType == nil {
		d.keyType = key.Type
	}
	if key.Type.ID() != d.keyType.ID() {
		return fmt.Errorf("key type changed from %s to %s", d.keyType, key.Type)
	}
	if err := d.microindex.Write(rec); err != nil {
		return err
	}
	return nil
}
