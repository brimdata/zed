package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

const zarExt = ".zar"

// TBD
type Indexer interface {
	Create(*zdx.Writer, zbuf.Reader)
	//Search([]byte) bool
}

//XXX this is a test stub that creates simple indexes of IP addresses
func CreateIndexes(dir string) error {
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
			err = IndexLogFile(path)
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

func IndexLogFile(path string) error {
	subdir := path + zarExt
	zdxName := "zdx:type:ip"
	zdxPath := filepath.Join(subdir, zdxName)
	// XXX remove without warning, should have force flag
	zdx.Remove(zdxPath)

	fmt.Printf("%s: indexing as %s\n", path, zdxPath)

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	reader, err := detector.LookupReader("bzng", file, resolver.NewContext())
	if err != nil {
		return err
	}
	table, err := indexTypeIP(reader)
	if err != nil {
		return err
	}
	if table.Size() == 0 {
		//XXX
		return errors.New("nothing to index")
	}
	// make subdirectory for index if it doesn't exist
	if err := os.Mkdir(subdir, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	framesize := 32 * 1024
	//XXX for now specify value size of 0, which means variable, but we always
	// write nil values.  we should change the implementation to allow key-only zdx files.
	writer, err := zdx.NewWriter(zdxPath, framesize, 0)
	if err != nil {
		return err
	}
	defer writer.Close()
	return zdx.Copy(writer, table)
}

func indexTypeIP(reader zbuf.Reader) (*zdx.MemTable, error) {
	table := zdx.NewMemTable()
	indexer := &TypeIndexer{Type: zng.TypeIP, Table: table}
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			return table, nil
		}
		indexer.record(rec.Type, rec.Raw)
	}
}
