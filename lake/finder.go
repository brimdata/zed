package lake

//XXX this should take a key range, consult the journal to get candidates,
// and return hits to segments.

/*
type findOptions struct {
	skipMissing bool
	zctx        *zson.Context
	addPath     func(lk *Lake, chunk chunk.Chunk, rec *zng.Record) (*zng.Record, error)
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
		cols := []zng.Column{
			{pathField, typ},
			{"first", zng.TypeTime},
			{"last", zng.TypeTime},
		}
		opt.addPath = func(lk *Lake, chunk chunk.Chunk, rec *zng.Record) (*zng.Record, error) {
			var path string
			if absolutePath {
				if lk.DataPath.Scheme == "file" {
					path = chunk.Path().Filepath()
				} else {
					path = chunk.Path().String()
				}
			} else {
				path = lk.Root.RelPath(chunk.Path())
			}
			return opt.zctx.AddColumns(rec, cols, []zng.Value{
				{cols[0].Type, zng.EncodeString(path)},
				{cols[1].Type, zng.EncodeTime(chunk.First)},
				{cols[2].Type, zng.EncodeTime(chunk.Last)},
			})
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
// access to the microindex files.
func Find(ctx context.Context, zctx *zson.Context, lk *Lake, query index.Query, hits chan<- *zng.Record, opts ...FindOption) error {
	opt := findOptions{
		zctx:    zctx,
		addPath: func(_ *Lake, _ chunk.Chunk, rec *zng.Record) (*zng.Record, error) { return rec, nil },
	}
	for _, o := range opts {
		if err := o(&opt); err != nil {
			return err
		}
	}
	defs, err := lk.ReadDefinitions(ctx)
	if err != nil {
		return err
	}
	matched, ok := defs.LookupQuery(query)
	if !ok {
		return zqe.ErrInvalid("no matching index rule found")
	}
	return Walk(ctx, lk, func(chunk chunk.Chunk) error {
		dir := chunk.ZarDir()
		reader, err := index.Find(ctx, zctx, dir, matched.DefID, matched.Values...)
		if err != nil {
			if zqe.IsNotFound(err) && opt.skipMissing {
				// No index for this rule.  Skip it if the skip boolean
				// says it's ok.  Otherwise, we return ErrNotExist since
				// the client was looking for something that wasn't indexed,
				// and they would probably want to know.
				return nil
			}
			return err
		}

		defer reader.Close()
		for {
			rec, err := reader.Read()
			if rec == nil || err != nil {
				return err
			}
			if rec, err = opt.addPath(lk, chunk, rec); err != nil {
				return err
			}

			select {
			case hits <- rec:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
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

func FindReadCloser(ctx context.Context, zctx *zson.Context, lk *Lake, query index.Query, opts ...FindOption) (zbuf.ReadCloser, error) {
	ctx, cancel := context.WithCancel(ctx)
	f := &findReadCloser{
		ctx:    ctx,
		cancel: cancel,
		hits:   make(chan *zng.Record),
	}
	go func() {
		f.err = Find(ctx, zctx, lk, query, f.hits, opts...)
		close(f.hits)
	}()
	return f, nil
}
*/
