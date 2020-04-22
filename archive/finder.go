package archive

import (
	"fmt"
	"os"
	"path/filepath"

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
func Find(dir string, rule Rule, pattern string, hits chan string) error {
	//XXX this should be parallelized with some locking presuming a little
	// parallelism won't mess up the file system assumptions
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%q: %v", path, err)
		}
		name := info.Name()
		if info.IsDir() {
			if filepath.Ext(name) == zarExt {
				//XXX need to merge into or replace existing index
				return filepath.SkipDir
			}
			// descend...
			return nil
		}
		// XXX should be regex 
		if filepath.Ext(name) == ".zng" {
			hit, err := SearchFile(path, pattern)
			if err != nil {
				fmt.Printf("%s\n", err)
			}
			if hit && hits != nil {
				hits <- path
			}
		}
		return nil
	})
	return err
}

func Search(path string, rule Rule, pattern string) (bool, error) {
	subdir, err := archiveDir(path)
	if err != nil {
		return false, err
	}
	finder := rule.NewFinder(subdir)
	keyType, err := finder.Open()
	if err != nil {
		if err == os.ErrNotExist {
			err = nil
		} else {
			err = fmt.Errorf("%s: %w", finder.Path(), err)
		}
		return false, err
	}
	keyBytes, err := keyType.Parse([]byte(pattern))
	if err != nil {
		return false, fmt.Errorf("%s: %w", finder.Path(), err)
	}
	rec, err := finder.Lookup(zng.Value{keyType, keyBytes})
	if err != nil {
		err = fmt.Errorf("%s: %w", finder.Path(), err)
	}
	return rec != nil, err
}
