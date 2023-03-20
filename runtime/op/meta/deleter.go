package meta

import (
	"errors"
	"sync"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/data"
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
	batches     []zbuf.Batch
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
	// Each time pull is called, we scan a whole object and see if any deletes
	// happen in it, then return each batch of the modified object.  If the
	// object doesn't have any deletions, we ignore it and keep scanning till
	// we find one.
	for {
		if len(d.batches) != 0 {
			batch := d.batches[0]
			d.batches = d.batches[1:]
			return batch, nil
		}
		batches, err := d.nextDeletion()
		if batches == nil || err != nil {
			d.close(err)
			return nil, err
		}
		d.batches = batches
	}
}

func (d *Deleter) nextDeletion() ([]zbuf.Batch, error) {
	var object *data.Object
	var batches []zbuf.Batch
	var scanner zbuf.Puller
	var count uint64
	for {
		if scanner == nil {
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
				if len(vals) != 1 {
					// We currently support only one partition per batch.
					return nil, errors.New("system error: meta.Deleter encountered multi-valued batch")
				}
			}
			scanner, object, err = newScanner(d.octx.Context, d.octx.Zctx, d.pool, d.unmarshaler, d.pruner, d.filter, d.progress, &vals[0])
			if err != nil {
				d.close(err)
				return nil, err
			}
			count = 0
		}
		batch, err := scanner.Pull(false)
		if err != nil {
			return nil, err
		}
		if batch != nil {
			count += uint64(len(batch.Values()))
			batches = append(batches, batch)
			continue
		}
		if count != object.Count {
			// This object had values deleted from it.  Record it to the
			// map and return the modified batches downstream to be written
			// to new objects.
			d.deleteObject(object.ID)
			return batches, nil
		}
		scanner = nil
		object = nil
		count = 0
		batches = batches[:0]
	}
}

func (d *Deleter) close(err error) {
	d.err = err
	d.done = true
}

func (d *Deleter) deleteObject(id ksuid.KSUID) {
	d.deletes.Store(id, nil)
}
