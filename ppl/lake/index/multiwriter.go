package index

import (
	"context"
	"path"

	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zqe"
	"go.uber.org/multierr"
)

type MultiWriter []*Writer

func NewMultiWriter(ctx context.Context, d iosrc.URI, defs []*Definition) (MultiWriter, error) {
	mkdirOnce := true
	writers := make(MultiWriter, 0, len(defs))
	for _, def := range defs {
	again:
		w, err := NewWriter(ctx, IndexPath(d, def.ID), def)
		if err != nil {
			if zqe.IsExists(err) {
				// skip indices that already exist
				continue
			}

			// Only mkdir once, if for some reason the fails again return error.
			if zqe.IsNotFound(err) && mkdirOnce {
				if err = mkdir(d); err == nil {
					mkdirOnce = false
					goto again
				}
			}

			writers.Abort()
			return nil, err
		}

		writers = append(writers, w)
	}
	return writers, nil
}

func (ws MultiWriter) Write(rec *zng.Record) error {
	for _, w := range ws {
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (ws MultiWriter) Close() (merr error) {
	for _, w := range ws {
		if err := w.Close(); err != nil {
			merr = multierr.Append(merr, err)
		}
	}
	return
}

func (ws MultiWriter) Indices() []Index {
	indices := make([]Index, len(ws))
	for i, w := range ws {
		u := w.URI
		u.Path = path.Dir(u.Path)
		indices[i] = Index{w.Definition, u}
	}
	return indices
}

func (ws MultiWriter) Abort() {
	for _, w := range ws {
		w.Abort()
	}
}
