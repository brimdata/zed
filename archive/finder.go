package archive

import (
	"fmt"
	"os"

	"github.com/brimsec/zq/zng"
)

// Find descends a directory hierarchy looking for index files that can
// resolve the provided rule.  For each such index found, the rule is used to
// see if the pattern is in the index.  If the pattern is in the index,
// then the path name of the file corresponding to that index is included in
// the slice of strings comprising the return value.
// XXX We currently allow only one pattern at a time though it might
// be more efficient at large scale to allow multipe patterns that
// are effectively OR-ed together so that there is locality of
// access to the zdx files.
func Find(dir string, rule Rule, pattern string, hits chan<- string, skipMissing bool) error {
	//XXX this should be parallelized with some locking presuming a little
	// parallelism won't mess up the file system assumptions
	return Walk(dir, func(zardir string) error {
		hit, err := Search(zardir, rule, pattern, skipMissing)
		if err != nil {
			return err
		}
		if hit != nil && hits != nil {
			hits <- ZarDirToLog(zardir)
		}
		return nil
	})
}

func FindZng(dir string, rule Rule, pattern string, hits chan<- *zng.Record, skipMissing bool) error {
	//XXX this should be parallelized with some locking presuming a little
	// parallelism won't mess up the file system assumptions
	return Walk(dir, func(zardir string) error {
		hit, err := Search(zardir, rule, pattern, skipMissing)
		if err != nil {
			return err
		}
		if hit != nil {
			hit.Keep()
			hits <- hit
		}
		return nil
	})
}

func Search(zardir string, rule Rule, pattern string, skipMissing bool) (*zng.Record, error) {
	finder := rule.NewFinder(zardir)
	keyType, err := finder.Open()
	if err != nil {
		if err == os.ErrNotExist && skipMissing {
			// No index for this rule.  Skip it if the skip boolean
			// says it's ok.  Otherwise, we return ErrNotExist since
			// the client was looking for something that wasn't indexed,
			// and they would probably want to know.
			err = nil
		} else {
			err = fmt.Errorf("%s: %w", finder.Path(), err)
		}
		return nil, err
	}
	defer finder.Close()
	if keyType == nil {
		// This happens when an index exists but is empty.
		return nil, nil
	}
	keyBytes, err := keyType.Parse([]byte(pattern))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", finder.Path(), err)
	}
	rec, err := finder.Lookup(zng.Value{keyType, keyBytes})
	if err != nil {
		err = fmt.Errorf("%s: %w", finder.Path(), err)
	}
	return rec, err
}
