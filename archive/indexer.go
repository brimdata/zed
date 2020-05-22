package archive

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"go.uber.org/zap"
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

func runZql(zardir string, rule *ZqlRule, progress chan<- string) error {
	logPath := ZarDirToLog(zardir)
	file, err := fs.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()
	zctx := resolver.NewContext()
	r := zngio.NewReader(file, zctx)
	fgi, err := NewFlowgraphIndexer(zctx, rule.Path(zardir), rule.keys, rule.framesize)
	if err != nil {
		return err
	}
	defer fgi.Close()
	out, err := driver.Compile(context.TODO(), rule.proc, r, false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}
	if progress != nil {
		progress <- fmt.Sprintf("%s: creating index %s", logPath, rule.Path(zardir))
	}
	return driver.Run(out, fgi, nil)
}

func run(zardir string, rules []Rule, progress chan<- string) error {
	logPath := ZarDirToLog(zardir)
	var writers []zbuf.WriteCloser
	for _, rule := range rules {
		if zrule, ok := rule.(*ZqlRule); ok {
			err := runZql(zardir, zrule, progress)
			if err != nil {
				return err
			}
			continue
		}
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

type FlowgraphIndexer struct {
	zctx *resolver.Context
	w    *zdx.Writer
}

func NewFlowgraphIndexer(zctx *resolver.Context, path string, keys []string, framesize int) (*FlowgraphIndexer, error) {
	if len(keys) == 0 {
		keys = []string{"key"}
	}
	writer, err := zdx.NewWriter(zctx, path, keys, framesize)
	if err != nil {
		return nil, err
	}
	return &FlowgraphIndexer{zctx, writer}, nil
}

func (f *FlowgraphIndexer) Write(_ int, batch zbuf.Batch) error {
	for i := 0; i < batch.Length(); i++ {
		if err := f.w.Write(batch.Index(i)); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (f *FlowgraphIndexer) Close() error {
	return f.w.Close()
}

func (f *FlowgraphIndexer) Warn(warning string) error          { return nil }
func (f *FlowgraphIndexer) Stats(stats api.ScannerStats) error { return nil }
func (f *FlowgraphIndexer) ChannelEnd(cid int) error           { return nil }
