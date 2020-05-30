package archive

import (
	"context"
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
	return "zdx-type-" + t.String()
}

func fieldZdxName(fieldname string) string {
	return "zdx-field-" + fieldname
}

func IndexDirTree(ark *Archive, rules []Rule, path string, progress chan<- string) error {
	return Walk(ark, func(zardir string) error {
		logPath := Localize(zardir, path)
		return run(zardir, rules, logPath, progress)
	})
}

func runOne(zardir string, rule Rule, inputPath string, progress chan<- string) error {
	file, err := fs.Open(inputPath)
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
	out, err := driver.CompileCustom(context.TODO(), &compiler{}, rule.proc, r, false, nano.MaxSpan, zap.NewNop())
	if err != nil {
		return err
	}
	if progress != nil {
		progress <- fmt.Sprintf("%s: creating index %s", inputPath, rule.Path(zardir))
	}
	return driver.Run(out, fgi, nil)
}

func run(zardir string, rules []Rule, logPath string, progress chan<- string) error {
	for _, rule := range rules {
		err := runOne(zardir, rule, logPath, progress)
		if err != nil {
			return err
		}
	}
	return nil
}

type FlowgraphIndexer struct {
	zctx *resolver.Context
	w    *zdx.Writer
}

func NewFlowgraphIndexer(zctx *resolver.Context, path string, keys []string, framesize int) (*FlowgraphIndexer, error) {
	if len(keys) == 0 {
		keys = []string{keyName}
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
