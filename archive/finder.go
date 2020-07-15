package archive

import (
	"context"
	"fmt"
	"os"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

type findOptions struct {
	skipMissing bool
	zctx        *resolver.Context
	addPath     func(ark *Archive, si SpanInfo, rec *zng.Record) (*zng.Record, error)
}

type FindOption func(*findOptions) error

func SkipMissing() FindOption {
	return func(opt *findOptions) error {
		opt.skipMissing = true
		return nil
	}
}

const DefaultAddPathField = "_log"

func AddPath(pathField string, absolutePath bool) FindOption {
	if pathField == "" {
		panic("missing pathField argument")
	}
	// Create a type alias called "zfile" that the client will
	// understand as the pathname of a zng file, e.g., available
	// as a new space via zqd.
	return func(opt *findOptions) error {
		typ, err := opt.zctx.LookupTypeAlias("zfile", zng.TypeString)
		if err != nil {
			return err
		}
		pathCol := []zng.Column{{pathField, typ}}
		opt.addPath = func(ark *Archive, si SpanInfo, rec *zng.Record) (*zng.Record, error) {
			var path string
			if absolutePath {
				path = si.LogID.Path(ark).String()
			} else {
				path = string(si.LogID)
			}
			val := zng.Value{pathCol[0].Type, zng.EncodeString(path)}
			return opt.zctx.AddColumns(rec, pathCol, []zng.Value{val})
		}
		return nil
	}
}

// Find descends a directory hierarchy looking for index files to search and
// for each such index, it executes the given query. If the pattern matches,
// a zng.Record of that row in the index is streamed to the hits channel.
// If multiple rows match, they are streamed in the order encountered in the index.
// Multiple records can match for a multi-key search where one or more keys are
// unspecified, implying a "don't care" condition for the unspecified sub-key(s).
// If the SkipMissing option is used, then log files that do not have the indicated index
// are silently skipped; otherwise, an error is returned for the missing index file
// and the search is terminated.  If the AddPath option is used, then a column is added
// to each returned zng.Record with the path of the log file relating that hit
// where the column name is given by the pathField argument.  If the passed-in
// ctx is canceled, the search will terminate with error context.Canceled.
// XXX We currently allow only one multi-key pattern at a time though it might
// be more efficient at large scale to allow multipe patterns that
// are effectively OR-ed together so that there is locality of
// access to the zdx files.
func Find(ctx context.Context, ark *Archive, query IndexQuery, hits chan<- *zng.Record, opts ...FindOption) error {
	opt := findOptions{
		zctx:    resolver.NewContext(),
		addPath: func(_ *Archive, _ SpanInfo, rec *zng.Record) (*zng.Record, error) { return rec, nil },
	}
	for _, o := range opts {
		if err := o(&opt); err != nil {
			return err
		}
	}
	indexInfo, ok := ark.indexes[query.indexName]
	if !ok {
		return zqe.E(zqe.NotFound)
	}
	return SpanWalk(ark, func(si SpanInfo, zardir iosrc.URI) error {
		searchHits := make(chan *zng.Record)
		var searchErr error
		go func() {
			defer close(searchHits)
			searchErr = search(ctx, opt.zctx, searchHits, zardir.AppendPath(indexInfo.Path), query.patterns)
			if searchErr != nil && os.IsNotExist(searchErr) && opt.skipMissing {
				// No index for this rule.  Skip it if the skip boolean
				// says it's ok.  Otherwise, we return ErrNotExist since
				// the client was looking for something that wasn't indexed,
				// and they would probably want to know.
				searchErr = nil
			}
		}()
		for hit := range searchHits {
			h, err := opt.addPath(ark, si, hit)
			if err != nil {
				return err
			}
			select {
			case hits <- h:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return searchErr
	})
}

func search(ctx context.Context, zctx *resolver.Context, hits chan<- *zng.Record, uri iosrc.URI, patterns []string) error {
	finder := zdx.NewFinder(zctx, uri)
	if err := finder.Open(); err != nil {
		return fmt.Errorf("%s: %w", finder.Path(), err)
	}
	defer finder.Close()
	keys, err := finder.ParseKeys(patterns)
	if err != nil {
		return fmt.Errorf("%s: %w", finder.Path(), err)
	}
	err = finder.LookupAll(ctx, hits, keys)
	if err != nil {
		err = fmt.Errorf("%s: %w", finder.Path(), err)
	}
	return err
}

type findReadCloser struct {
	ctx    context.Context
	cancel context.CancelFunc
	hits   chan *zng.Record
	err    error
}

func (f *findReadCloser) Read() (*zng.Record, error) {
	select {
	case r, ok := <-f.hits:
		if !ok {
			return nil, f.err
		}
		return r, nil
	case <-f.ctx.Done():
		return nil, f.ctx.Err()
	}
}

func (f *findReadCloser) Close() error {
	f.cancel()
	return nil
}

func FindReadCloser(ctx context.Context, ark *Archive, query IndexQuery, opts ...FindOption) (zbuf.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	f := &findReadCloser{
		ctx:    ctx,
		cancel: cancel,
		hits:   make(chan *zng.Record),
	}
	go func() {
		f.err = Find(ctx, ark, query, f.hits, opts...)
		close(f.hits)
	}()
	return f, nil
}
