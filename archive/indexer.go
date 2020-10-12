package archive

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/proc/cut"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
)

// XXX Embedding the type and field names like this can result in some clunky
// file names. We might want to re-work the naming scheme.

func typeMicroIndexName(t zng.Type) string {
	return "microindex-type-" + t.String() + ".zng"
}

func fieldMicroIndexName(fieldname string) string {
	return "microindex-field-" + fieldname + ".zng"
}

func IndexDirTree(ctx context.Context, ark *Archive, rules []Rule, path string, progress chan<- string) error {
	return Walk(ctx, ark, func(chunk Chunk) error {
		zardir := chunk.ZarDir(ark)
		logPath := chunk.Localize(ark, path)
		return run(ctx, zardir, rules, logPath, progress)
	})
}

func runOne(ctx context.Context, zardir iosrc.URI, rule Rule, inputPath iosrc.URI, progress chan<- string) error {
	rc, err := iosrc.NewReader(ctx, inputPath)
	if err != nil {
		return err
	}
	defer rc.Close()
	zctx := resolver.NewContext()
	r := zngio.NewReader(rc, zctx)
	fgi, err := NewFlowgraphIndexer(ctx, zctx, rule.Path(zardir), rule.keys, rule.framesize)
	if err != nil {
		return err
	}
	if progress != nil {
		progress <- fmt.Sprintf("%s: creating index %s", inputPath, rule.Path(zardir))
	}
	err = driver.Run(ctx, fgi, rule.proc, zctx, r, driver.Config{
		Custom: compile,
	})
	if err != nil {
		fgi.Abort()
		return err
	}
	return fgi.Close()
}

func run(ctx context.Context, zardir iosrc.URI, rules []Rule, logPath iosrc.URI, progress chan<- string) error {
	for _, rule := range rules {
		if err := runOne(ctx, zardir, rule, logPath, progress); err != nil {
			return err
		}
	}
	return nil
}

type FlowgraphIndexer struct {
	zctx    *resolver.Context
	w       *microindex.Writer
	keyType zng.Type
	cutter  *cut.Cutter
}

func NewFlowgraphIndexer(ctx context.Context, zctx *resolver.Context, uri iosrc.URI, keys []field.Static, framesize int) (*FlowgraphIndexer, error) {
	if len(keys) == 0 {
		keys = []field.Static{keyName}
	}
	writer, err := microindex.NewWriterWithContext(ctx, zctx, uri.String(), keys, framesize)
	if err != nil {
		return nil, err
	}
	fields, resolvers := expr.CompileAssignments(keys, keys)
	return &FlowgraphIndexer{
		zctx:   zctx,
		w:      writer,
		cutter: cut.NewStrictCutter(zctx, false, fields, resolvers),
	}, nil
}

func (f *FlowgraphIndexer) Write(_ int, batch zbuf.Batch) error {
	defer batch.Unref()
	for i := 0; i < batch.Length(); i++ {
		rec := batch.Index(i)
		key, err := f.cutter.Cut(rec)
		if err != nil {
			return fmt.Errorf("checking index record: %w", err)
		}
		if f.keyType == nil {
			f.keyType = key.Type
		}
		if key.Type.ID() != f.keyType.ID() {
			return fmt.Errorf("key type changed from %s to %s", f.keyType, key.Type)
		}
		if err := f.w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (f *FlowgraphIndexer) Close() error {
	return f.w.Close()
}

func (f *FlowgraphIndexer) Abort() error {
	return f.w.Abort()
}

func (f *FlowgraphIndexer) Warn(warning string) error          { return nil }
func (f *FlowgraphIndexer) Stats(stats api.ScannerStats) error { return nil }
func (f *FlowgraphIndexer) ChannelEnd(cid int) error           { return nil }
