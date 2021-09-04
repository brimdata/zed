package index

import (
	"context"

	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	//"github.com/segmentio/ksuid"
)

//XXX we should store the seek index range as an option in the index object
// and this could return that range instead of a bool

func Search(ctx context.Context, engine storage.Engine, path *storage.URI, value zng.Value) (bool, error) {
	finder, err := index.NewFinder(ctx, zson.NewContext(), engine, path)
	if err != nil {
		return false, err
	}
	defer finder.Close()
	//XXX just pass value to Lookup method instead of return a record
	// that is passed back in
	keys, err := finder.WrapKeys(value)
	if err != nil {
		return false, err
	}
	// XXX we should get rid of the hits channel and just return the matched
	// records.  The parallelism model that zar used is different here and
	// parallelism can be worked out above in the scheduler.
	hits := make(chan *zng.Record)
	var searchErr error
	go func() {
		searchErr = finder.LookupAll(ctx, hits, keys)
		close(hits)
	}()
	var found bool
	for range hits {
		found = true
	}
	if searchErr != nil {
		err = searchErr
	}
	return found, err
}
