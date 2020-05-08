package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Find descends a directory hierarchy looking for index files to search and
// for each such index, it looks up the given pattern.  If the pattern matches,
// a zng.Record of that row in the index is streamed to the hits channel.
// If multiple rows match, they are streamed in the order encountered in the index.
// Multiple records can match for a multi-key search where one or more keys are
// unspecified, implying a "don't care" condition for the unspecified sub-key(s).
// If skipMissing is true, then log files that do not have the indicated index
// are silently skipped; otherwise, an error is returned for the missing index file
// and the search is terminated.  If pathField is non-zero, then a column is added
// to each returned zng.Record with the path of the log file relating that hit
// where the column name is given by the pathField argument.  If the passed-in
// ctx is canceled, the search will terminate with error context.Canceled.
// XXX We currently allow only one pattern at a time though it might
// be more efficient at large scale to allow multipe patterns that
// are effectively OR-ed together so that there is locality of
// access to the zdx files.
func Find(ctx context.Context, rootDir, indexName string, pattern []string, hits chan<- *zng.Record, pathField string, skipMissing bool) error {
	//XXX this can be parallelized fairly easily since the search results are
	// share the same type context allowing the zng.Records to easily comprise
	// a single stream
	zctx := resolver.NewContext()
	var pathCol []zng.Column
	if pathField != "" {
		// Create a type alias called "zfile" that the client will
		// understand as the pathname of a zng file, e.g., available
		// as a new space via zqd.
		typ, err := zctx.LookupTypeAlias("zfile", zng.TypeString)
		if err != nil {
			return err
		}
		pathCol = []zng.Column{{pathField, typ}}
	}
	ctx, cancel := context.WithCancel(ctx)
	return Walk(rootDir, func(zardir string) error {
		path := filepath.Join(zardir, indexName)
		searchHits := make(chan *zng.Record)
		var searchErr error
		go func() {
			searchErr = search(ctx, zctx, searchHits, path, pattern, skipMissing)
			close(searchHits)
		}()
		logPath := ZarDirToLog(zardir)
		for hit := range searchHits {
			if pathCol != nil {
				val := zng.Value{pathCol[0].Type, zng.EncodeString(logPath)}
				var err error
				hit, err = zctx.AddColumns(hit, pathCol, []zng.Value{val})
				if err != nil {
					cancel()
					for _ = range searchHits {
						// let search unravel
					}
					return err
				}
			}
			hits <- hit
		}
		return searchErr
	})
}

func search(ctx context.Context, zctx *resolver.Context, hits chan<- *zng.Record, path string, pattern []string, skipMissing bool) error {
	finder := zdx.NewFinder(zctx, path)
	if err := finder.Open(); err != nil {
		if err == os.ErrNotExist && skipMissing {
			// No index for this rule.  Skip it if the skip boolean
			// says it's ok.  Otherwise, we return ErrNotExist since
			// the client was looking for something that wasn't indexed,
			// and they would probably want to know.
			err = nil
		} else {
			err = fmt.Errorf("%s: %w", finder.Path(), err)
		}
		return err
	}
	defer finder.Close()
	keys, err := finder.ParseKeys(pattern)
	if err != nil {
		return fmt.Errorf("%s: %w", finder.Path(), err)
	}
	err = finder.LookupAll(ctx, hits, keys)
	if err != nil {
		err = fmt.Errorf("%s: %w", finder.Path(), err)
	}
	return err
}
