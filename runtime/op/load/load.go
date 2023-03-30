package load

import (
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
    	octx   *op.Context
    	lk     *lake.Root
	parent zbuf.Puller
	pool   string
}

func New(octx *op.Context, lk *lake.Root, parent zbuf.Puller, pool string) *Op {
	return &Op{
		octx:   octx,
	    	lk:     lk,
		parent: parent,
		pool:   pool,
	}
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	if done {
        	b, err := o.parent.Pull(true)
        	if err != nil {
            		return nil, err
        	}
        	if b != nil {
            	panic("non-nil done batch")
        	}
        	return nil, nil
    	}
    	batch, err := o.parent.Pull(false)
 		if batch == nil || err != nil {
			return nil, err
		}
    	reader := zbuf.PullerReader(o.parent)
    	poolID, err := (o.lk).PoolID(o.octx.Context, o.pool)
    	if err != nil {
        	return nil, err
    	}
    	pool, err := (o.lk).OpenPool(o.octx.Context, poolID)
    	if err != nil {
           	return nil, err
    	}
    	branch, err := pool.OpenBranchByName(o.octx.Context, "main")
    	if err != nil {
        	return nil, err
    	}
    	_, err_load := branch.Load(o.octx.Context, o.octx.Zctx, reader, "", "", "")
    	//_, err_load := branch.Load(o.octx.Context, o.octx.Zctx, reader, o.params["author"], o.params["message"], o.params["meta"])// so Params be a map[string]string
    	if err_load != nil {
        	return nil, err_load
    	}
    	return zbuf.NewBatch(batch, batch.Values()), nil //is this what we want to return?
}
