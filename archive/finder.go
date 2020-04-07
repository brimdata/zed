package archive

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/sst"
)

func Find(dir string, pattern []byte) ([]string, error) {
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
			hit, err := SearchFile(path, pattern)
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

//XXX for now we hard code search for IP address.
func SearchFile(path string, pattern []byte) (bool, error) {
	subdir := path + zarExt
	sstName := "sst:type:ip" //XXX
	sstPath := filepath.Join(subdir, sstName)
	finder, err := sst.NewFinder(sstPath)
	if err != nil {
		if err == os.ErrNotExist {
			err = nil
		} else {
			err = fmt.Errorf("%s: %s", sstPath, err)
		}
		return false, err
	}
	v, err := finder.Lookup(pattern)
	if err != nil {
		err = fmt.Errorf("%s: %s", sstPath, err.Error())
	}
	return v != nil, err
}
