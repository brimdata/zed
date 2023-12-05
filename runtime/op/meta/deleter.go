package meta

import (
	"errors"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Deleter struct {
	parent      zbuf.Puller
	scanner     zbuf.Puller
	filter      zbuf.Filter
	pruner      expr.Evaluator
	octx        *op.Context
	pool        *lake.Pool
	progress    *zbuf.Progress
	unmarshaler *zson.UnmarshalZNGContext
	done        bool
	err         error
	deletes     *sync.Map
}

func NewDeleter(octx *op.Context, parent zbuf.Puller, pool *lake.Pool, filter zbuf.Filter, pruner expr.Evaluator, progress *zbuf.Progress, deletes *sync.Map) *Deleter {
	return &Deleter{
		parent:      parent,
		filter:      filter,
		pruner:      pruner,
		octx:        octx,
		pool:        pool,
		progress:    progress,
		unmarshaler: zson.NewZNGUnmarshaler(),
		deletes:     deletes,
	}
}

func (d *Deleter) Pull(done bool) (zbuf.Batch, error) {
	if d.done {
		return nil, d.err
	}
	if done {
		if d.scanner != nil {
			_, err := d.scanner.Pull(true)
			d.close(err)
			d.scanner = nil
		}
		return nil, d.err
	}
	for {
		if d.scanner == nil {
			scanner, err := d.nextDeletion()
			if scanner == nil || err != nil {
				d.close(err)
				return nil, err
			}
			d.scanner = scanner
		}
		if batch, err := d.scanner.Pull(false); err != nil {
			d.close(err)
			return nil, err
		} else if batch != nil {
			return batch, nil
		}
		d.scanner = nil
	}
}

func (d *Deleter) nextDeletion() (zbuf.Puller, error) {
	for {
		if d.parent == nil { //XXX
			return nil, nil
		}
		// Pull the next object to be scanned.  It must be an object
		// not a partition.
		batch, err := d.parent.Pull(false)
		if batch == nil || err != nil {
			return nil, err
		}
		vals := batch.Values()
		if len(vals) != 1 {
			// We currently support only one partition per batch.
			return nil, errors.New("internal error: meta.Deleter encountered multi-valued batch")
		}
		if hasDeletes, err := d.hasDeletes(&vals[0]); err != nil {
			return nil, err
		} else if !hasDeletes {
			continue
		}
		// Use a no-op progress so stats are not inflated.
		var progress zbuf.Progress
		scanner, object, err := newScanner(d.octx.Context, d.octx.Zctx, d.pool, d.unmarshaler, d.pruner, d.filter, &progress, &vals[0])
		if err != nil {
			return nil, err
		}
		d.deleteObject(object.ID)
		return scanner, nil
	}
}

func (d *Deleter) hasDeletes(val *zed.Value) (bool, error) {
	scanner, object, err := newScanner(d.octx.Context, d.octx.Zctx, d.pool, d.unmarshaler, d.pruner, d.filter, d.progress, val)
	if err != nil {
		return false, err
	}
	var count uint64
	for {
		batch, err := scanner.Pull(false)
		if err != nil {
			return false, err
		}
		if batch == nil {
			return count != object.Count, nil
		}
		count += uint64(len(batch.Values()))
	}
}

func (d *Deleter) close(err error) {
	d.err = err
	d.done = true
}

func (d *Deleter) deleteObject(id ksuid.KSUID) {
	d.deletes.Store(id, nil)
}
