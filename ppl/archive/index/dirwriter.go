package index

import (
	"context"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/multierr"
)

type DirWriter []*Writer

func NewDirWriter(ctx context.Context, d Dir, defs []*Def) (DirWriter, error) {
	var writers DirWriter
	for _, def := range defs {
		w, err := d.newIndexWriter(ctx, def)
		if err != nil {
			if zqe.IsExists(err) {
				// skip indices that already exist
				continue
			}
			writers.Abort()
			return nil, err
		}
		writers = append(writers, w)
	}
	return writers, nil
}

func (ws DirWriter) WriteBatch(batch zbuf.Batch) error {
	for _, w := range ws {
		batch.Ref()
		if err := w.WriteBatch(batch); err != nil {
			return err
		}
	}
	return nil
}

func (ws DirWriter) Write(rec *zng.Record) error {
	for _, w := range ws {
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (ws DirWriter) Close() (merr error) {
	for _, w := range ws {
		if err := w.Close(); err != nil {
			merr = multierr.Append(merr, err)
		}
	}
	// Drop all indices if there was an error closing. All for one, one for all.
	if merr != nil {
		ws.Abort()
	}
	return
}

func (ws DirWriter) Abort() {
	for _, w := range ws {
		w.Abort()
	}
}
