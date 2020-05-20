package archive

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zbuf"
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

func IndexDirTree(ark *Archive, rules []Rule, progress chan<- string) error {
	nerr := 0
	return Walk(ark, func(zardir string) error {
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
	var writers []zbuf.WriteCloser
	for _, rule := range rules {
		w, err := rule.NewIndexer(zardir)
		if err != nil {
			return err
		}
		writers = append(writers, w)
		if progress != nil {
			progress <- fmt.Sprintf("%s: creating index %s", logPath, rule.Path(zardir))
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
		for _, w := range writers {
			if err := w.Write(rec); err != nil {
				return err
			}
		}
	}
	var lastErr error
	// XXX this loop could be parallelized.. the close on the index writers
	// dump the in-memory table to disk.  Also, we should use a zql proc
	// graph to do this work instead of the in-memory map once we have
	// group-by spills working.
	for _, w := range writers {
		if err := w.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
