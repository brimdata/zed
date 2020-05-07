package archive

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/zngio"
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

// Indexer provides a means to index a zng file.  First, a stream of zng.Records
// is written to the Indexer via zbuf.Writer, then the indexed records are read
// as a stream via zbuf.Reader.  The index is managed as a zdx bundle.
// XXX currently we are supporting just in-memory indexing but it would be
// straightforward to extend this to spill in-memory tables then merge them
// on close a la LSM.
type Indexer interface {
	zbuf.Reader
	zbuf.Writer
	Path() string
}

func IndexDirTree(dir string, rules []Rule, progress chan<- string) error {
	nerr := 0
	return Walk(dir, func(zardir string) error {
		if err := run(zardir, rules, progress); err != nil {
			if progress != nil {
				progress <- fmt.Sprintf("%s: %s\n", zardir, err)
			}
			nerr++
			if nerr > 10 {
				//XXX
				return errors.New("stopping after too many errors...")
			}
		}
		return nil
	})
}

func run(zardir string, rules []Rule, progress chan<- string) error {
	logPath := ZarDirToLog(zardir)
	var indexers []Indexer
	for _, rule := range rules {
		indexer, err := rule.NewIndexer(zardir)
		if err != nil {
			return err
		}
		indexers = append(indexers, indexer)
		if progress != nil {
			progress <- fmt.Sprintf("%s: creating index %s", logPath, indexer.Path())
		}
	}
	file, err := fs.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := zngio.NewReader(file, resolver.NewContext())
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
			if err := indexer.Write(rec); err != nil {
				return err
			}
		}
	}
	// we make the framesize here larger than the writer framesize
	// since the writer always writes a bit past the threshold
	const framesize = 32 * 1024 * 2
	// XXX this loop could be parallelized
	for _, indexer := range indexers {
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
