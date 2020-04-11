package archive

import (
	"fmt"
	"os"
	"path/filepath"
)

// Find descends a directory hierarchy looking for the provided pattern
// that is used by the rule to determine whether the pattern exists
// in each index found that conforms with the rule.
// XXX We currently allow only one pattern at a time though it might
// be more efficient at large scale to allow multipe patterns that
// are effectively OR-ed together so that there is locality of
// access to the zdx files.
func Find(dir string, rule Rule, pattern string) ([]string, error) {
	//XXX this should be parallelized with some locking presuming a little
	// parallelism won't mess up the file system assumptions
	var hits []string
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
		if filepath.Ext(name) == ".bzng" {
			hit, err := Search(path, rule, pattern)
			if err != nil {
				fmt.Printf("%s\n", err)
			}
			if hit {
				hits = append(hits, path)
			}
		}
		return nil
	})
	return hits, err
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
			err = fmt.Errorf("%s: %s", finder.Path(), err)
		}
		return false, err
	}
	keyBytes, err := keyType.Parse([]byte(pattern))
	if err != nil {
		return false, fmt.Errorf("%s: %s", finder.Path(), err.Error())
	}
	rec, err := finder.Lookup(keyBytes)
	if err != nil {
		err = fmt.Errorf("%s: %s", finder.Path(), err.Error())
	}
	return rec != nil, err
}
