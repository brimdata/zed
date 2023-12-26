package vam

import (
	"errors"
	"fmt"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"golang.org/x/sync/errgroup"
)

type Puller interface {
	Pull(done bool) (vector.Any, error)
}

// XXX need a semaphore pattern here we scanner can run ahead and load objects
// and vectors concurrently but be limited by the semaphore so there is a reasonable
// amount of locality while still being highly parallel.

// project (pull from downstream) => iterator
// filter+project (pull from downstram)
// agg
// filter+agg->project-partials (pull from group-by)

type VecScanner struct {
	parent      zbuf.Puller
	pruner      expr.Evaluator
	octx        *op.Context
	pool        *lake.Pool
	once        sync.Once
	paths       []field.Path
	cache       *vcache.Cache
	progress    *zbuf.Progress
	unmarshaler *zson.UnmarshalZNGContext
	resultCh    chan result
	doneCh      chan struct{}
}

func NewVecScanner(octx *op.Context, cache *vcache.Cache, parent zbuf.Puller, pool *lake.Pool, paths []field.Path, pruner expr.Evaluator, progress *zbuf.Progress) *VecScanner {
	return &VecScanner{
		cache:       cache,
		octx:        octx,
		parent:      parent,
		pruner:      pruner,
		pool:        pool,
		paths:       paths,
		progress:    progress,
		unmarshaler: zson.NewZNGUnmarshaler(),
		doneCh:      make(chan struct{}),
		resultCh:    make(chan result),
	}
}

// XXX we need vector scannerstats and means to update them here.

// XXX change this to pull/load vector by each type within an object and
// return an object containing the overall projection, which might be a record
// or could just be a single vector.  the downstream operator has to be
// configured to expect it, e.g., project x:=a.b,y:=a.b.c (like cut but in vspace)
// this would be Record{x:(proj a.b),y:(proj:a.b.c)} so the elements would be
// single fields.  For each object/type that matches the projection we would make
// a Record vec and let GC reclaim them.  Note if a col is missing, it's a constant
// vector of error("missing").

func (v *VecScanner) Pull(done bool) (vector.Any, error) {
	v.once.Do(func() {
		// Block p.ctx's cancel function until p.run finishes its
		// cleanup.
		v.octx.WaitGroup.Add(1)
		go v.run()
	})
	if done {
		select {
		case v.doneCh <- struct{}{}:
			return nil, nil
		case <-v.octx.Done():
			return nil, v.octx.Err()
		}
	}
	if r, ok := <-v.resultCh; ok {
		return r.vector, r.err
	}
	return nil, v.octx.Err()
}

func (v *VecScanner) run() {
	defer func() {
		v.octx.WaitGroup.Done()
	}()
	for {
		//XXX should make an object puller that wraps this...
		batch, err := v.parent.Pull(false)
		if batch == nil || err != nil {
			v.sendResult(nil, err)
			return
		}
		vals := batch.Values()
		if len(vals) != 1 {
			// We require exactly one data object per pull.
			err := errors.New("system error: VecScanner encountered multi-valued batch")
			v.sendResult(nil, err)
			return
		}
		named, ok := vals[0].Type.(*zed.TypeNamed)
		if !ok {
			v.sendResult(nil, fmt.Errorf("system error: VecScanner encountered unnamed object: %s", zson.String(vals[0])))
			return
		}
		if named.Name != "data.Object" {
			v.sendResult(nil, fmt.Errorf("system error: VecScanner encountered unnamed object: %q", named.Name))
			return
		}
		var meta data.Object
		if err := v.unmarshaler.Unmarshal(&vals[0], &meta); err != nil {
			v.sendResult(nil, fmt.Errorf("system error: VecScanner could not unmarshal value: %q", zson.String(vals[0])))
			return
		}
		object, err := v.cache.Fetch(v.octx.Context, meta.VectorURI(v.pool.DataPath), meta.ID)
		if err != nil {
			v.sendResult(nil, err)
			return
		}
		if err := v.genVecs(object, v.resultCh); err != nil {
			v.sendResult(nil, err)
			return
		}
	}
}

func (v *VecScanner) sendResult(vec vector.Any, err error) (bool, bool) {
	select {
	case v.resultCh <- result{vec, err}:
		return false, true
	case <-v.doneCh:
		if vec != nil {
			vec.Unref() //XXX add
		}
		b, pullErr := v.parent.Pull(true)
		if err == nil {
			err = pullErr
		}
		if err != nil {
			select {
			case v.resultCh <- result{err: err}:
				return true, false
			case <-v.octx.Done():
				return false, false
			}
		}
		if b != nil {
			b.Unref()
		}
		return true, true
	case <-v.octx.Done():
		return false, false
	}
}

type result struct {
	vector vector.Any
	err    error
}

// XXX for each type that has target columns, we return a bundle of the vectors.
// this will usually just be one bundle but for eclectic data, could be
// a bundle per relevant type.  Note that each slot has a unique type so the
// the bundles are interleaved but non-overlapping in terms of their output slots.
func (v *VecScanner) genVecs(o *vcache.Object, ch chan result) error {
	//XXX we should map the type to a shared context and have a table to
	// memoize the per-type lookup so we don't have to spin through every type?
	var group errgroup.Group
	for typeKey := range o.Types() {
		typeKey := uint32(typeKey)
		for _, path := range v.paths {
			path := path
			group.Go(func() error {
				vec, err := o.Load(typeKey, path)
				//XXX for now ignore error, e.g., need to distinguish between
				// missing/ignore and other real errors
				if err != nil {
					err = nil
				}
				if vec == nil || err != nil {
					return err
				}
				v.sendResult(vec, nil)
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return err
		}
	}
	return nil
}
