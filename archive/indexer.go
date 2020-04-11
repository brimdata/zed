package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

const zarExt = ".zar"

// XXX Embedding the type and field names like this can result in some clunky
// file names. We might want to re-work the naming scheme.

func typeZdxName(t zng.Type) string {
	return "zdx:type:" + t.String()
}

func fieldZdxName(fieldname string) string {
	return "zdx:field:" + fieldname
}

func archiveDir(path string) (string, error) {
	//XXX for now the index directory is the name of the zng file
	// with the ".zar" extension
	subdir := path + zarExt
	// make subdirectory for index if it doesn't exist
	if err := os.Mkdir(subdir, 0755); err != nil {
		if !os.IsExist(err) {
			return "", err
		}
	}
	return subdir, nil
}

// Indexer provides a means to index a zng file.  First, a stream of zng.Records
// is written to the Indexer via zbuf.Writer, then the indexed records are read
// as a stream via zbuf.Reader.   The index is managed as a zdx bundle.
// XXX currently we are supporting just in-memory indexing but it would be
// straightforward to extend this to spill in-memory tables then merge them
// on close a la LSM.
type Indexer interface {
	Path() string
	zbuf.Writer
	zbuf.Reader
}

func IndexDirTree(dir string, rules []Rule) error {
	nerr := 0
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
			err = Run(path, rules)
			if err != nil {
				fmt.Printf("%s: %s\n", path, err)
				nerr++
				if nerr > 10 {
					//XXX
					return errors.New("stopping after too many errors...")
				}
			}
			// drop through and continue
		}
		return nil
	})
	return err
}

func Run(path string, rules []Rule) error {
	subdir, err := archiveDir(path)
	if err != nil {
		return err
	}
	var indexers []Indexer
	for _, rule := range rules {
		indexer := rule.NewIndexer(subdir)
		indexers = append(indexers, indexer)
		fmt.Printf("%s: creating index %s\n", path, indexer.Path())
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bzngio.NewReader(file, resolver.NewContext())
	// XXX This for-loop could be easily parallelized by having each writer
	// live in its own go routine and sending the rec over a set of
	// blocking channels (so we flow-control it).
	for {
		rec, err := reader.Read()
		if err != nil {
			return err
		}
		if rec == nil {
			break
		}
		for _, indexer := range indexers {
			err := indexer.Write(rec)
			if err != nil {
				return err
			}
		}
	}
	// XXX this loop could be parallelized
	for _, indexer := range indexers {
		const framesize = 32 * 1024 // XXX
		writer, err := zdx.NewWriter(indexer.Path(), framesize)
		if err != nil {
			return err
		}
		if err := zbuf.Copy(writer, indexer); err != nil {
			writer.Close()
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
	}
	return nil
}
