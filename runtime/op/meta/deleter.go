package meta

import (
	"io"
	"sync"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Deleter struct {
	parent      zbuf.Puller
	current     zbuf.Puller
	filter      zbuf.Filter
	pctx        *op.Context
	pool        *lake.Pool
	progress    *zbuf.Progress
	snap        commits.View
	unmarshaler *zson.UnmarshalZNGContext
	done        bool
	err         error
	deletes     *sync.Map
}

// XXX shouldn't pass in snap
func NewDeleter(pctx *op.Context, parent zbuf.Puller, pool *lake.Pool, snap commits.View, filter zbuf.Filter, progress *zbuf.Progress, deletes *sync.Map) *Deleter {
	return &Deleter{
		parent:      parent,
		filter:      filter,
		pctx:        pctx,
		pool:        pool,
		progress:    progress,
		snap:        snap,
		unmarshaler: zson.NewZNGUnmarshaler(),
		deletes:     deletes,
	}
}

func (d *Deleter) Pull(done bool) (zbuf.Batch, error) {
	if d.done {
		return nil, d.err
	}
	if done {
		if d.current != nil {
			_, err := d.current.Pull(true)
			d.close(err)
			d.current = nil
		}
		return nil, d.err
	}
	for {
		if d.current == nil {
			if d.parent == nil { //XXX
				d.close(nil)
				return nil, nil
			}
			// Pull the next partition from the parent snapshot and
			// set up the next scanner to pull from.
			var part Partition
			ok, err := nextPartition(d.parent, &part, d.unmarshaler)
			if !ok || err != nil {
				d.close(err)
				return nil, err
			}
			d.current, err = newDeleterScanner(d, part)
			if err != nil {
				d.close(err)
				return nil, err
			}
		}
		batch, err := d.current.Pull(false)
		if err != nil {
			d.close(err)
			return nil, err
		}
		if batch != nil {
			return batch, nil
		}
		d.current = nil
	}
}

func (d *Deleter) close(err error) {
	d.err = err
	d.done = true
}

func (d *Deleter) deleteObject(id ksuid.KSUID) {
	d.deletes.Store(id, nil)
}

func newDeleterScanner(d *Deleter, part Partition) (zbuf.Puller, error) {
	pullers := make([]zbuf.Puller, 0, len(part.Objects))
	pullersDone := func() {
		for _, puller := range pullers {
			puller.Pull(true)
		}
	}
	for _, o := range part.Objects {
		//XXX not sure this is right... filter applies here?
		rg, err := objectRange(d.pctx.Context, d.pool, d.snap, d.filter, o)
		if err != nil {
			return nil, err
		}
		rc, err := o.NewReader(d.pctx.Context, d.pool.Storage(), d.pool.DataPath, rg)
		if err != nil {
			pullersDone()
			return nil, err
		}
		scanner, err := zngio.NewReader(d.pctx.Zctx, rc).NewScanner(d.pctx.Context, d.filter)
		if err != nil {
			pullersDone()
			rc.Close()
			return nil, err
		}
		pullers = append(pullers, &deleteScanner{
			object:  o,
			scanner: scanner,
			closer:  rc,
			deleter: d,
		})
	}
	if len(pullers) == 1 {
		return pullers[0], nil
	}
	return merge.New(d.pctx.Context, pullers, lake.ImportComparator(d.pctx.Zctx, d.pool).Compare), nil
}

type deleteScanner struct {
	object  *data.Object
	once    sync.Once
	batches []zbuf.Batch
	deleter *Deleter
	scanner zbuf.Scanner
	closer  io.Closer
	err     error
}

func (d *deleteScanner) Pull(done bool) (zbuf.Batch, error) {
	//XXX why sync.Once?  This is not concurrent
	d.once.Do(func() {
		for {
			batch, err := d.scanner.Pull(done)
			if batch == nil || err != nil {
				d.deleter.progress.Add(d.scanner.Progress())
				if err2 := d.closer.Close(); err == nil {
					err = err2
				}
				d.err = err
				var count uint64
				for _, b := range d.batches {
					count += uint64(len(b.Values()))
				}
				if count == d.object.Count {
					// Only return batches from objects where values have been
					// deleted.
					d.batches = nil
				} else {
					d.deleter.deleteObject(d.object.ID)
				}
				return
			}
			d.batches = append(d.batches, batch)
		}
	})
	if len(d.batches) == 0 || d.err != nil {
		return nil, d.err
	}
	batch := d.batches[0]
	d.batches = d.batches[1:]
	return batch, nil
}
